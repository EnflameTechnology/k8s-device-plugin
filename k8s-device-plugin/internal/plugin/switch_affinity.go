// Copyright (c) 2024, ENFLAME INC.  All rights reserved.
// EF_SUPPORT

package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	"enflame/k8s-device-plugin/internal/eflib/interfaces"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/types"
	"enflame/k8s-device-plugin/internal/utils"
)

type PcieSwitchAffinity struct {
	eflib          interfaces.HandleInterface
	switchTopoList []SwitchTopo
	podResource    *PodResource
}

type SwitchTopo struct {
	Device1 DeviceTopo
	Device2 DeviceTopo
	Score   int
}

type DeviceTopo struct {
	Index    *int   `json:"index,omitempty"`
	Name     string `json:"name,omitempty"`
	Busid    string `json:"busid,omitempty"`
	TreePath string `json:"pcieTree,omitempty"`
}

type AllocatedRecord struct {
	PodName       string `json:"podName,omitempty"`
	PodUid        string `json:"uuid,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
}

const (
	MaxRetryTimes      = 30
	AssignedContainers = "k8s-device-plugin/assignedContainers"
)

func NewPcieSwitchAffinity(eflib interfaces.HandleInterface, resourceName string) (*PcieSwitchAffinity, error) {
	podResource, err := NewPodResource(resourceName)
	if err != nil {
		return nil, err
	}
	switchTopoList, err := MakeSwitchTopoList(eflib)
	if err != nil {
		return nil, err
	}
	return &PcieSwitchAffinity{
		eflib:          eflib,
		switchTopoList: switchTopoList,
		podResource:    podResource,
	}, nil
}

func (rer *PcieSwitchAffinity) AllocateByPcieSwitchAffinity(reqDeviceCount int) ([]string, error) {
	// var allocateIds []string
	// for index, req := range reqs.ContainerRequests {
	// 	reqDeviceIds := req.DevicesIDs
	// 	logger.Info("container %d Request GCU count: %d", index, len(reqDeviceIds))
	assumePod, assignedMap, err := rer.podResource.GetAssumePod(reqDeviceCount)
	if err != nil {
		return nil, err
	}
	allocateIds, err := rer.selectDeviceIds(reqDeviceCount)
	if err != nil {
		return nil, err
	}
	logger.Info("allocate device: %v for container by pcie switch affinity", allocateIds)
	if err := rer.podResource.PatchPodAnnotations(assumePod, assignedMap, allocateIds); err != nil {
		return nil, err
	}

	return allocateIds, nil
}

func (rer *PcieSwitchAffinity) selectDeviceIds(reqDeviceCount int) ([]string, error) {
	allocatedMap, err := rer.getAllocatedMap()
	if err != nil {
		return nil, err
	}

	availableDeviceList := rer.getAvailableDeviceList(allocatedMap)
	if len(availableDeviceList) < reqDeviceCount {
		err := fmt.Errorf("resource %s is insufficient, request %d, but available is %d", rer.podResource.ResourceName,
			reqDeviceCount, len(availableDeviceList))
		logger.Error(err)
		return nil, err
	}

	if reqDeviceCount == 1 {
		return rer.selectSingleCardMinAffinity(availableDeviceList)
	}

	deviceCombineList := make([][]int, 0)
	rer.combine(availableDeviceList, reqDeviceCount, 0, &deviceCombineList, []int{})
	maxScore, selectedCombine := -1, []int{}
	for _, deviceCombine := range deviceCombineList {
		deviceCombineScore := rer.getDeviceCombineScore(deviceCombine)
		if maxScore < 0 || deviceCombineScore > maxScore {
			maxScore = deviceCombineScore
			selectedCombine = deviceCombine
		}
	}
	selectedDevices := []string{}
	for _, devID := range selectedCombine {
		selectedDevices = append(selectedDevices, fmt.Sprintf("%d", devID))
	}
	return selectedDevices, nil
}

func (rer *PcieSwitchAffinity) getAvailableDeviceList(allocatedMap map[int]AllocatedRecord) []int {
	availableDeviceList := []int{}
	for deviceId, devicePath := range rer.eflib.GetDeviceIDPathMap() {
		if _, ok := allocatedMap[deviceId]; ok {
			continue
		}
		if !utils.FileIsExist(devicePath) {
			logger.Info("Warn: device %s not exist when select card for container, skip it", devicePath)
			continue
		}
		availableDeviceList = append(availableDeviceList, deviceId)
	}
	sort.Ints(availableDeviceList)
	logger.Info("list of currently available gcu resources list: %v", availableDeviceList)
	return availableDeviceList
}

func (rer *PcieSwitchAffinity) getDeviceCombineScore(deviceCombine []int) int {
	score := 0
	devicePairList := make([][]int, 0)
	rer.combine(deviceCombine, 2, 0, &devicePairList, []int{})
	for _, devicePair := range devicePairList {
		name1 := rer.eflib.DeviceGetHandleByIndex(devicePair[0]).GetName()
		name2 := rer.eflib.DeviceGetHandleByIndex(devicePair[1]).GetName()
		for _, switchTopo := range rer.switchTopoList {
			if switchTopo.Device1.Name == name1 && switchTopo.Device2.Name == name2 ||
				(switchTopo.Device1.Name == name2 && switchTopo.Device2.Name == name1) {
				score += switchTopo.Score
				break
			}
		}
	}
	return score
}

func (rer *PcieSwitchAffinity) combine(availableDeviceList []int, reqDeviceCount int, start int, result *[][]int, current []int) {
	if len(current) == reqDeviceCount {
		temp := make([]int, len(current))
		copy(temp, current)
		*result = append(*result, temp)
		return
	}

	for i := start; i < len(availableDeviceList); i++ {
		current = append(current, availableDeviceList[i])
		rer.combine(availableDeviceList, reqDeviceCount, i+1, result, current)
		current = current[:len(current)-1]
	}
}

func (rer *PcieSwitchAffinity) selectSingleCardMinAffinity(availableDeviceList []int) ([]string, error) {
	minScore, selectdDevice := -1, -1
	for _, deviceId := range availableDeviceList {
		devName := rer.eflib.DeviceGetHandleByIndex(deviceId).GetName()
		deviceAffinityMaxScore := rer.getDeviceAffinityMaxScore(deviceId, availableDeviceList)
		if deviceAffinityMaxScore < 0 {
			err := fmt.Errorf("device %s is not found in switchTopoList: %v", devName, rer.switchTopoList)
			logger.Error(err)
			return nil, err
		}
		if minScore < 0 || deviceAffinityMaxScore < minScore {
			minScore = deviceAffinityMaxScore
			selectdDevice = deviceId
		}
	}
	if selectdDevice == -1 {
		err := fmt.Errorf("resource %s is insufficient, request 1, but available is 0", rer.podResource.ResourceName)
		logger.Error(err)
		return nil, err
	}
	return []string{fmt.Sprintf("%d", selectdDevice)}, nil
}

func (rer *PcieSwitchAffinity) getDeviceAffinityMaxScore(deviceId int, availableDeviceList []int) int {
	deviceAffinityMaxScore := -1
	for _, switchTopo := range rer.switchTopoList {
		if !utils.IntList(availableDeviceList).ContainSubList([]int{*switchTopo.Device1.Index, *switchTopo.Device2.Index}) {
			continue
		}
		if !utils.IntList([]int{*switchTopo.Device1.Index, *switchTopo.Device2.Index}).ContainValue(deviceId) {
			continue
		}
		if deviceAffinityMaxScore < 0 || switchTopo.Score > deviceAffinityMaxScore {
			deviceAffinityMaxScore = switchTopo.Score
		}
	}
	return deviceAffinityMaxScore
}

func (rer *PcieSwitchAffinity) getAllocatedMap() (map[int]AllocatedRecord, error) {
	allocatedRecordMap := map[int]AllocatedRecord{}
	selector := fields.SelectorFromSet(fields.Set{"spec.nodeName": rer.podResource.NodeName})
	var podList *corev1.PodList
	var err error
	for i := 0; i < MaxRetryTimes; i++ {
		podList, err = rer.podResource.ClientSet.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
			FieldSelector: selector.String(),
			LabelSelector: labels.Everything().String(),
		})
		if err != nil {
			logger.Error(err, "list all pods in node %s failed, retry times: %d", rer.podResource.NodeName, i)
			time.Sleep(time.Second * 1)
			continue
		}
		break
	}
	if err != nil {
		logger.Error(err, "list all pods in node %s failed in max retry times: %d", rer.podResource.NodeName, MaxRetryTimes)
		return nil, err
	}
	for _, pod := range podList.Items {
		if !rer.podResource.checkPodRequestGCU(&pod) || pod.Spec.NodeName != rer.podResource.NodeName {
			continue
		}
		assignedString, ok := pod.Annotations[AssignedContainers]
		if !ok {
			continue
		}
		clusterAssignedMap := map[string][]string{}
		if err := json.Unmarshal([]byte(assignedString), &clusterAssignedMap); err != nil {
			logger.Error(err, "unmarshal assigned containers to map of pod: %s failed, content: %s", pod.Name, assignedString)
			return nil, err
		}
		for containerName, deviceIds := range clusterAssignedMap {
			for _, deviceID := range deviceIds {
				intID, err := strconv.Atoi(deviceID)
				if err != nil {
					logger.Error(err, "convert string type device id: %s to int failed", deviceID)
					return nil, err
				}
				if _, ok := allocatedRecordMap[intID]; ok {
					err := fmt.Errorf("device %d is assigned repeat, allocatedRecordMap: %v", intID, allocatedRecordMap)
					logger.Error(err)
					return nil, err
				}
				allocatedRecordMap[intID] = AllocatedRecord{
					PodName:       pod.Name,
					PodUid:        string(pod.UID),
					ContainerName: containerName,
				}
			}
		}
	}
	logger.Info("get allocatedRecordMap success, content: %v", utils.ConvertToString(allocatedRecordMap))
	return allocatedRecordMap, nil
}

func MakeSwitchTopoList(eflib interfaces.HandleInterface) ([]SwitchTopo, error) {
	switchTopoList := []SwitchTopo{}
	// init
	deviceIds := sortDeviceIds(eflib.GetDeviceIDPathMap())
	devTopoList := []DeviceTopo{}
	for _, devId := range deviceIds {
		devTopo := &DeviceTopo{
			Index:    &devId,
			Name:     eflib.DeviceGetHandleByIndex(devId).GetName(),
			Busid:    "",
			TreePath: "",
		}
		if err := devTopo.getDeviceBusId(eflib.DeviceGetHandleByIndex(devId).GetName()); err != nil {
			return nil, err
		}
		if devTopo.Busid == "" {
			logger.Warn("resource: %s not support pcie switch affinity scheduler", eflib.DeviceGetHandleByIndex(devId).DeviceType())
			return switchTopoList, nil
		}

		if err := devTopo.getDevicePcieTree(); err != nil {
			return nil, err
		}
		logger.Info("build device %d topo: %v", devId, utils.ConvertToString(*devTopo))
		devTopoList = append(devTopoList, *devTopo)
	}

	// calculate affinity
	for i := 0; i < len(devTopoList)-1; i++ {
		device1 := devTopoList[i]
		for j := i + 1; j < len(devTopoList); j++ {
			device2 := devTopoList[j]
			score, err := calculateScore(device1.TreePath, device2.TreePath)
			if err != nil {
				return nil, err
			}
			switchTopoList = append(switchTopoList, SwitchTopo{
				Device1: device1,
				Device2: device2,
				Score:   score,
			})
		}
	}

	sort.Slice(switchTopoList, func(i, j int) bool {
		return switchTopoList[i].Score > switchTopoList[j].Score
	})
	// for rank, switchTopo := range switchTopoList {
	// 	logger.Info("rank %d, switch topo: %v", rank+1, switchTopo)
	// }
	return switchTopoList, nil
}

func sortDeviceIds(deviceIDPathMap map[int]string) []int {
	deviceIds := []int{}
	for devId := range deviceIDPathMap {
		deviceIds = append(deviceIds, devId)
	}
	sort.Ints(deviceIds)
	return deviceIds
}

func calculateScore(treeCur, treeNext string) (int, error) {
	bridgeList1 := strings.Split(treeCur, "/")
	bridgeList2 := strings.Split(treeNext, "/")
	if len(bridgeList1) < 4 || len(bridgeList2) < 4 {
		err := fmt.Errorf("pcie tree is invalid: %s and %s", treeCur, treeNext)
		logger.Error(err)
		return 0, err
	}

	minLength := func(x, y int) int {
		if x < y {
			return x
		}
		return y
	}(len(bridgeList1[3:]), len(bridgeList2[3:]))

	score := 0
	for i := 0; i < minLength; i++ {
		if bridgeList1[3:][i] == bridgeList2[3:][i] {
			score += 1
		} else {
			break
		}
	}
	return score, nil
}

func (rer *DeviceTopo) getDeviceBusId(devName string) error {
	busIdPath := fmt.Sprintf("%s/%s", types.BusIDPrefix, devName)
	content, err := os.ReadFile(busIdPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn("device busid file: %s not exist", busIdPath)
			return nil
		}
		logger.Error(err, "get gcu device %d busId failed", devName)
		return err
	}
	rer.Busid = string(bytes.TrimSpace(content))
	return nil
}

// getDevicePcieTree return pcie tree for device busid
func (rer *DeviceTopo) getDevicePcieTree() error {
	cmd := exec.Command("readlink", "-f", fmt.Sprintf("%s/%s", types.DevicePciPrefix, rer.Busid))
	output, err := cmd.Output()
	if err != nil {
		logger.Error(err, "get pcie tree failed for busid: %s", rer.Busid)
		return err
	}
	rer.TreePath = string(bytes.TrimSpace(output))
	return nil
}
