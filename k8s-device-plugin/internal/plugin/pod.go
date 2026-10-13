// Copyright (c) 2024, ENFLAME INC.  All rights reserved.
// EF_SUPPORT

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"enflame/k8s-device-plugin/internal/informer"
	"enflame/k8s-device-plugin/internal/kube"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/utils"
)

type PodResource struct {
	ClientSet    *kubernetes.Clientset
	NodeName     string
	ResourceName string
	// pendingSrc, when non-nil, supplies this node's pending pods from an
	// alternative source (informer cache / kubelet) instead of a per-call
	// apiserver List. nil => apiserver List (default). Set via SetPendingPodSource.
	pendingSrc pendingPodLister
	// pendingListRV is the ResourceVersion used on the per-Allocate/reconcile
	// pod List (apiserver pod-source). "0" (default) serves the List from the
	// apiserver watch cache (v1.22 indexes spec.nodeName so it is O(this node's
	// pods), no etcd hit). Empty string falls back to a strongly-consistent etcd
	// quorum read (O(cluster total pods)); set via PENDING_LIST_RESOURCE_VERSION.
	pendingListRV string
}

func NewPodResource(resourceName string) (*PodResource, error) {
	clientSet, err := kube.GetKubeClient()
	if err != nil {
		return nil, err
	}
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		err = fmt.Errorf("please set env NODE_NAME")
		logger.Error(err)
		return nil, err
	}
	return &PodResource{
		ClientSet:     clientSet,
		NodeName:      nodeName,
		ResourceName:  resourceName,
		pendingListRV: pendingListResourceVersion(),
	}, nil
}

// pendingListResourceVersion returns the ResourceVersion for the pending/running
// pod List. Only two settings are supported:
//
//	"0" (default) : served from the apiserver watch cache (cheap, O(this node)
//	                 via the spec.nodeName index in v1.22, no etcd hit).
//	"etcd" / ""   : strongly-consistent etcd quorum read (O(cluster total
//	                 pods); only if a strictly up-to-date view is required).
//
// Any other value is rejected and falls back to the "0" default.
func pendingListResourceVersion() string {
	v, ok := os.LookupEnv("PENDING_LIST_RESOURCE_VERSION")
	if !ok {
		return "0" // default: watch-cache read
	}
	switch strings.TrimSpace(v) {
	case "0":
		return "0"
	case "etcd", "":
		return "" // strongly-consistent etcd read
	default:
		logger.Warn("invalid PENDING_LIST_RESOURCE_VERSION %q (want \"0\" or \"etcd\"), using default \"0\"", v)
		return "0"
	}
}

func (rer *PodResource) GetAssumePod(reqCount int) (*corev1.Pod, map[string][]string, error) {
	pods, err := rer.GetCandidatePods()
	if err != nil {
		return nil, nil, err
	}

	for _, pod := range pods {
		logger.Info("candidate pod %s in ns %s request GCU count: %d with timestamp %s(%d)", pod.Name, pod.Namespace,
			rer.GetPodReqestDeviceCount(pod), pod.CreationTimestamp, pod.CreationTimestamp.Unix())
		containerName, assignedMap, err := rer.PodExistContainerCanBind(reqCount, pod)
		if err != nil {
			return nil, nil, err
		}
		if containerName == "" {
			continue
		}
		assignedMap[containerName] = []string{}
		logger.Info("found assumed pod %s in ns %s container: %s request device count %d", pod.Name, pod.Namespace,
			containerName, reqCount)
		return pod, assignedMap, nil
	}
	err = fmt.Errorf("can not found assumed pod request device count %d", reqCount)
	logger.Error(err)
	return nil, nil, err
}

func (rer *PodResource) PodExistContainerCanBind(reqCount int, pod *corev1.Pod) (string, map[string][]string, error) {
	assignedMap := map[string][]string{}
	assignedContainers, ok := pod.Annotations[AssignedContainers]
	if ok {
		if err := json.Unmarshal([]byte(assignedContainers), &assignedMap); err != nil {
			logger.Error(err, "unmarshal assigned containers to map of pod: %s failed, content: %s", pod.Name, assignedContainers)
			return "", nil, err
		}
	}
	for _, container := range pod.Spec.Containers {
		if val, ok := container.Resources.Limits[corev1.ResourceName(rer.ResourceName)]; ok {
			if assigned, ok := assignedMap[container.Name]; ok && len(assigned) > 0 {
				continue
			}
			if int(val.Value()) == reqCount {
				logger.Info("container: %s of pod: %s can allocated gcu device count: %d", container.Name,
					pod.Name, reqCount)
				return container.Name, assignedMap, nil
			}
		}
	}
	logger.Info("candidate pod: %s no container can allocated gcu device count: %d, skip this pod", pod.Name, reqCount)
	return "", nil, nil
}

// Get GCU Memory of the Pod
func (rer *PodResource) GetPodReqestDeviceCount(pod *corev1.Pod) int {
	total := 0
	for _, container := range pod.Spec.Containers {
		if val, ok := container.Resources.Limits[corev1.ResourceName(rer.ResourceName)]; ok {
			total += int(val.Value())
		}
	}
	return total
}

func (rer *PodResource) PatchPodAnnotations(assumePod *corev1.Pod, assignedMap map[string][]string,
	allocateIds []string) error {
	for key, value := range assignedMap {
		if len(value) == 0 {
			assignedMap[key] = allocateIds
			break
		}
	}

	values, err := json.Marshal(assignedMap)
	if err != nil {
		logger.Error(err, "marshal assigned containers to string of pod: %s failed, content: %v", assumePod.Name, assignedMap)
		return err
	}
	patchAnnotations := map[string]interface{}{
		"metadata": map[string]map[string]string{"annotations": {
			AssignedContainers: string(values),
		}}}
	patchedAnnotationBytes, err := json.Marshal(patchAnnotations)
	if err != nil {
		logger.Error(err, "marshal patch annotations failed, content: %v", patchAnnotations)
		return err
	}
	for i := 0; i < MaxRetryTimes; i++ {
		_, err = rer.ClientSet.CoreV1().Pods(assumePod.Namespace).Patch(context.TODO(), assumePod.Name,
			k8sTypes.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
		if err == nil {
			logger.Info("patch pod %s annotations: %s success", assumePod.Name, string(patchedAnnotationBytes))
			break
		}
		logger.Error(err, "patch pod %s annotations: %s failed, retry times: %d", assumePod.Name,
			string(patchedAnnotationBytes), i)
		time.Sleep(time.Second * 1)
	}
	if err != nil {
		logger.Error(err, "patch pod %s annotations: %s failed in max retry time: %d", assumePod.Name,
			string(patchedAnnotationBytes), MaxRetryTimes)
		return err
	}

	for i := 0; i < MaxRetryTimes; i++ {
		clusterPod, err := rer.ClientSet.CoreV1().Pods(assumePod.Namespace).Get(context.TODO(), assumePod.Name,
			metav1.GetOptions{})
		if err != nil {
			logger.Error(err, "get pod %s from cluster failed, retry times: %d", assumePod.Name, i)
			time.Sleep(time.Second * 1)
			continue
		}
		if !rer.checkPodPatchFinish(clusterPod, assignedMap) {
			logger.Warn("check cluster pod %s patch not finish, retry times: %d", clusterPod.Name, i)
			time.Sleep(time.Second * 1)
			continue
		}
		return nil
	}
	err = fmt.Errorf("check patch pod %s annotations: %s failed in max retry time: %d", assumePod.Name,
		string(patchedAnnotationBytes), MaxRetryTimes)
	logger.Error(err)
	return err
}

func (rer *PodResource) checkPodPatchFinish(clusterPod *corev1.Pod, assignedMap map[string][]string) bool {
	assignedString, ok := clusterPod.Annotations[AssignedContainers]
	if !ok {
		logger.Warn("cluster pod has no field: %s", AssignedContainers)
		return false
	}
	clusterAssignedMap := map[string][]string{}
	if err := json.Unmarshal([]byte(assignedString), &clusterAssignedMap); err != nil {
		logger.Error(err, "unmarshal assigned containers to map of pod: %s failed, content: %s",
			clusterPod.Name, assignedString)
		return false
	}
	if len(assignedMap) != len(clusterAssignedMap) {
		logger.Warn("expect assigned annotations map: %v is different cluster pod assigned annotations map: %v",
			assignedMap, clusterAssignedMap)
		return false
	}
	for key, expect := range assignedMap {
		actual, ok := clusterAssignedMap[key]
		if !ok || len(actual) != len(expect) {
			logger.Warn("expect assigned annotations map: %v is different cluster pod assigned annotations map: %v",
				assignedMap, clusterAssignedMap)
			return false
		}
		for index, id := range expect {
			if actual[index] != id {
				logger.Warn("expect assigned annotations map: %v is different cluster pod assigned annotations map: %v",
					assignedMap, clusterAssignedMap)
				return false
			}
		}
	}
	logger.Info("check cluster pod %s patch assigned annotations finish: %v", clusterPod.Name, assignedMap)
	return true
}

// pick up the current node pod with status is pending
func (rer *PodResource) GetCandidatePods() ([]*corev1.Pod, error) {
	candidatePods := []*corev1.Pod{}
	allPods, err := rer.GetPendingPodsInNode()
	if err != nil {
		return nil, err
	}
	for i := range allPods {
		if rer.checkPodRequestGCU(&allPods[i]) {
			candidatePods = append(candidatePods, &allPods[i])
		}
	}
	return makePodOrderdByAge(candidatePods), nil
}

func (rer *PodResource) GetPendingPodsInNode() ([]corev1.Pod, error) {
	pods := []corev1.Pod{}
	podIDMap := map[k8sTypes.UID]bool{}

	// Source the raw node pod list. Default is a direct apiserver List (its
	// field selector already restricts to this node + Pending). An alternative
	// source (informer cache / kubelet) may return all phases, so the loop below
	// also guards on Pending — a no-op for the apiserver path.
	var rawPods []corev1.Pod
	if rer.pendingSrc != nil {
		srcPods, err := rer.pendingSrc.ListPendingPods()
		if err != nil {
			return nil, err
		}
		rawPods = srcPods
	} else {
		podList, err := rer.GetPodListsByListAPIServer()
		if err != nil {
			return nil, err
		}
		rawPods = podList.Items
	}

	for _, pod := range rawPods {
		if pod.Spec.NodeName != rer.NodeName {
			logger.Warn("Pod name %s in ns %s is not assigned to node %s as expected, it's placed on node %s, skip it",
				pod.Name, pod.Namespace, rer.NodeName, pod.Spec.NodeName)
			continue
		}
		if pod.Status.Phase != corev1.PodPending {
			continue
		}
		logger.Info("list pod %s in ns %s in node %s with status is %s", pod.Name, pod.Namespace,
			rer.NodeName, pod.Status.Phase)
		if _, ok := podIDMap[pod.UID]; !ok {
			pods = append(pods, pod)
			podIDMap[pod.UID] = true
		}
	}
	return pods, nil
}

// SetPendingPodSource configures where GetPendingPodsInNode reads this node's
// pending pods from (large-scale optimization, design §5.5):
//   - "" / "apiserver": direct apiserver List (default, no change).
//   - "informer":       node-scoped SharedInformer local cache (one watch,
//     reuses the supplied PodInformer); requires a started informer.
//   - "kubelet":        node-local kubelet /pods (zero apiserver).
func (rer *PodResource) SetPendingPodSource(source string, podInformer *informer.PodInformer) error {
	switch strings.TrimSpace(source) {
	case "", PodSourceApiServer:
		rer.pendingSrc = nil
	case PodSourceInformer:
		if podInformer == nil {
			return fmt.Errorf("pod-source %q requires a running pod informer", PodSourceInformer)
		}
		rer.pendingSrc = &informerPendingSource{informer: podInformer, nodeName: rer.NodeName}
	case PodSourceKubelet:
		kc, err := newKubeletClient(rer.NodeName)
		if err != nil {
			return fmt.Errorf("init kubelet pod-source: %w", err)
		}
		rer.pendingSrc = &kubeletPendingSource{client: kc, nodeName: rer.NodeName}
	default:
		return fmt.Errorf("unknown pod-source %q (want apiserver|informer|kubelet)", source)
	}
	logger.Info("by-card pending-pod source for resource %s: %q", rer.ResourceName, source)
	return nil
}

func (rer *PodResource) GetPodListsByListAPIServer() (*corev1.PodList, error) {
	selector := fields.SelectorFromSet(fields.Set{"spec.nodeName": rer.NodeName, "status.phase": "Pending"})
	podList, err := rer.ClientSet.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		FieldSelector:   selector.String(),
		LabelSelector:   labels.Everything().String(),
		ResourceVersion: rer.pendingListRV,
	})
	for i := 0; i < MaxRetryTimes && err != nil; i++ {
		logger.Error(err, "failed to get pending pod list by apiserver, retry time: %d", i+1)
		podList, err = rer.ClientSet.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
			FieldSelector:   selector.String(),
			LabelSelector:   labels.Everything().String(),
			ResourceVersion: rer.pendingListRV,
		})
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		logger.Error(err, "get cluster pod list by apiserver failed in MaxRetryTimes: %d", MaxRetryTimes)
		return nil, err
	}
	return podList, nil
}

func (rer *PodResource) checkPodRequestGCU(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if val, ok := container.Resources.Limits[corev1.ResourceName(rer.ResourceName)]; ok && val.Value() > 0 {
			return true
		}
	}
	return false
}

func makePodOrderdByAge(pods []*corev1.Pod) []*corev1.Pod {
	newPodList := make(orderedPodCreatedTimestamp, 0, len(pods))
	for _, v := range pods {
		newPodList = append(newPodList, v)
	}
	sort.Sort(newPodList)
	logger.Info("candidatePods list after order by create time stamp: %s", utils.ConvertToString([]*corev1.Pod(newPodList)))
	return []*corev1.Pod(newPodList)
}

type orderedPodCreatedTimestamp []*corev1.Pod

func (rer orderedPodCreatedTimestamp) Len() int {
	return len(rer)
}

func (rer orderedPodCreatedTimestamp) Less(i, j int) bool {
	return rer[i].ObjectMeta.CreationTimestamp.Before(&rer[j].ObjectMeta.CreationTimestamp)
}

func (rer orderedPodCreatedTimestamp) Swap(i, j int) {
	rer[i], rer[j] = rer[j], rer[i]
}

// ---- by-card ("按调度结果挂卡") helpers ----

// GetAssumePodByAssumedTime is the by-card counterpart of GetAssumePod: it orders
// candidate pending pods by the scheduler assumed-time annotation (earliest
// first) instead of creation timestamp, so the i-th anonymous Allocate call is
// matched to the i-th scheduled pod. RISK-2.
func (rer *PodResource) GetAssumePodByAssumedTime(reqCount int, assumedTimeKey string) (*corev1.Pod, map[string][]string, error) {
	allPods, err := rer.GetPendingPodsInNode()
	if err != nil {
		return nil, nil, err
	}
	candidatePods := []*corev1.Pod{}
	for i := range allPods {
		if rer.checkPodRequestGCU(&allPods[i]) {
			candidatePods = append(candidatePods, &allPods[i])
		}
	}
	candidatePods = makePodOrderedByAssumedTime(candidatePods, assumedTimeKey)

	for _, pod := range candidatePods {
		t, ok := parseAssumedTime(pod, assumedTimeKey)
		logger.Info("candidate pod %s/%s request count: %d with assumed-time %d(valid=%v)",
			pod.Namespace, pod.Name, rer.GetPodReqestDeviceCount(pod), t, ok)
		containerName, assignedMap, err := rer.PodExistContainerCanBind(reqCount, pod)
		if err != nil {
			return nil, nil, err
		}
		if containerName == "" {
			continue
		}
		assignedMap[containerName] = []string{}
		logger.Info("found assumed pod %s/%s container: %s request device count %d by assumed-time",
			pod.Namespace, pod.Name, containerName, reqCount)
		return pod, assignedMap, nil
	}
	err = fmt.Errorf("can not found assumed pod request device count %d", reqCount)
	logger.Error(err)
	return nil, nil, err
}

// makePodOrderedByAssumedTime orders pods by the scheduler assumed-time annotation
// (nanosecond int64, earliest first). Missing/invalid sort last; ties break by UID.
func makePodOrderedByAssumedTime(pods []*corev1.Pod, assumedTimeKey string) []*corev1.Pod {
	sort.SliceStable(pods, func(i, j int) bool {
		ti, oki := parseAssumedTime(pods[i], assumedTimeKey)
		tj, okj := parseAssumedTime(pods[j], assumedTimeKey)
		if oki != okj {
			return oki // valid annotation sorts before missing/invalid
		}
		if oki && ti != tj {
			return ti < tj
		}
		return pods[i].UID < pods[j].UID
	})
	logger.Info("candidatePods list after order by assumed-time: %s", utils.ConvertToString(pods))
	return pods
}

// parseAssumedTime reads the scheduler assumed-time annotation as int64
// nanoseconds. The bool reports whether it is present and valid.
func parseAssumedTime(pod *corev1.Pod, assumedTimeKey string) (int64, bool) {
	v, ok := pod.Annotations[assumedTimeKey]
	if !ok {
		return 0, false
	}
	t, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		logger.Warn("pod %s/%s assumed-time annotation %q invalid: %v", pod.Namespace, pod.Name, v, err)
		return 0, false
	}
	return t, true
}

// ListRunningGCUPods returns the Running, non-terminating pods on this node that
// request the plugin's resource (used by the stock-pod reconcile).
func (rer *PodResource) ListRunningGCUPods() ([]*corev1.Pod, error) {
	selector := fields.SelectorFromSet(fields.Set{"spec.nodeName": rer.NodeName, "status.phase": string(corev1.PodRunning)})
	var podList *corev1.PodList
	var err error
	for i := 0; i < MaxRetryTimes; i++ {
		podList, err = rer.ClientSet.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
			FieldSelector:   selector.String(),
			LabelSelector:   labels.Everything().String(),
			ResourceVersion: rer.pendingListRV,
		})
		if err != nil {
			logger.Error(err, "list running pods in node %s failed, retry times: %d", rer.NodeName, i)
			time.Sleep(time.Second * 1)
			continue
		}
		break
	}
	if err != nil {
		return nil, err
	}
	pods := []*corev1.Pod{}
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Spec.NodeName != rer.NodeName || !rer.checkPodRequestGCU(pod) || isPodTerminal(pod) {
			continue
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

// PatchPodAnnotationKeys strategic-merge patches the given annotation keys onto
// the pod (only the listed keys are touched), with retry. Used by reconcile.
func (rer *PodResource) PatchPodAnnotationKeys(pod *corev1.Pod, annotations map[string]string) error {
	patch := map[string]interface{}{
		"metadata": map[string]map[string]string{"annotations": annotations},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal annotation patch: %w", err)
	}
	for i := 0; i < MaxRetryTimes; i++ {
		_, err = rer.ClientSet.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name,
			k8sTypes.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
		if err == nil {
			return nil
		}
		logger.Error(err, "patch pod %s/%s annotations %s failed, retry times: %d", pod.Namespace, pod.Name,
			string(patchBytes), i)
		time.Sleep(time.Second * 1)
	}
	return fmt.Errorf("patch pod %s/%s annotations failed in max retry %d: %w", pod.Namespace, pod.Name, MaxRetryTimes, err)
}

// isPodTerminal reports whether a pod no longer holds its devices.
func isPodTerminal(pod *corev1.Pod) bool {
	if pod.DeletionTimestamp != nil {
		return true
	}
	return pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed
}
