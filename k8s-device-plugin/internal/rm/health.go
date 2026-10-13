/*
 * Copyright (c) 2019-2022, NVIDIA CORPORATION.  All rights reserved.
 * Copyright (c) 2024, ENFLAME INC.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY Type, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rm

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"enflame/k8s-device-plugin/internal/informer"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/types"
)

const (
	// envDisableHealthChecks defines the environment variable that is checked to determine whether healthchecks
	// should be disabled. If this envvar is set to "all" or contains the string "xids", healthchecks are
	// disabled entirely. If set, the envvar is treated as a comma-separated list of Xids to ignore. Note that
	// this is in addition to the Application errors that are already ignored.
	envDisableHealthChecks = "DP_DISABLE_HEALTHCHECKS"
	// allHealthChecks        = "xids"
	// DEVICE_UNAVAILABLE = "0"
)

func watchHealth(stop <-chan interface{}, reportList chan<- []*Device, devices Devices, podInformer *informer.PodInformer) {
	var enableHealthCheck = true

	disableHealthChecks := strings.ToLower(os.Getenv(envDisableHealthChecks))
	if strings.Contains(disableHealthChecks, "all") {
		enableHealthCheck = false
		logger.Warn("health check disabled with env %s: %s", envDisableHealthChecks, disableHealthChecks)
		return
	}

	for {
		select {
		case <-stop:
			return
		default:
		}
		if enableHealthCheck {
			healthyStatusChangedList := []*Device{}
			for _, d := range devices {
				health := DeviceHealthCheck(d.PCIBusID)
				exists, podInfo := podInformer.IsDeviceInUse(d.ID)

				isHealthy := health == types.HEALTH_PASS
				currentlyHealthy := d.Health == pluginapi.Healthy

				switch {
				case !isHealthy && currentlyHealthy && !exists:
					healthyStatusChangedList = append(healthyStatusChangedList, d)
					logger.Warn("Health check error on Device=%s, the device will go unhealthy.", d.ID)

				case !isHealthy && currentlyHealthy && exists:
					logger.Warn("device %s is being used by pod: %v, can not report unhealth status to kubelet, skip it", d.ID, podInfo)

				case !isHealthy && !currentlyHealthy && exists:
					healthyStatusChangedList = append(healthyStatusChangedList, d)
					logger.Warn("Health check error on Device=%s, but it is being used by pod: %v, the device will go healthy.", d.ID, podInfo)

				case isHealthy && !currentlyHealthy:
					healthyStatusChangedList = append(healthyStatusChangedList, d)
					logger.Warn("Health check recover on Device=%s, the device will go healthy.", d.ID)
				}
			}
			if len(healthyStatusChangedList) > 0 {
				reportList <- healthyStatusChangedList
			}
		}
		//Sleep 5 second
		time.Sleep(5 * time.Second)
	}
}

func DeviceHealthCheck(busID string) types.DeviceState {
	checkPath := filepath.Join(types.PCIDevicePath, busID, "enable")
	enableBytes, err := os.ReadFile(checkPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn("health check for device pci path: %s not exist", checkPath)
			return types.HEALTH_FAIL
		}
		logger.Error(err, "read enable file: %s failed", checkPath)
		return types.HEALTH_FAIL
	}
	enable := strings.TrimSpace(string(enableBytes))
	if enable != "1" {
		logger.Warn("health check for device pci path: %s not enable", checkPath)
		return types.HEALTH_FAIL
	}
	return types.HEALTH_PASS
}

// EF_CLOSE
/*
// CheckHealth performs health checks on a set of devices, writing to the 'unhealthy' channel with any unhealthy devices
func (r *efmlResourceManager) checkHealth(stop <-chan interface{}, devices Devices, unhealthy chan<- *Device) error {
	disableHealthChecks := strings.ToLower(os.Getenv(envDisableHealthChecks))
	if disableHealthChecks == "all" {
		disableHealthChecks = allHealthChecks
	}
	if strings.Contains(disableHealthChecks, "xids") {
		return nil
	}

	ret := r.efml.Init()
	if ret != efml.SUCCESS {
		if *r.config.Flags.FailOnInitError {
			return fmt.Errorf("failed to initialize EFML: %v", ret)
		}
		return nil
	}
	defer func() {
		ret := r.efml.Shutdown()
		if ret != efml.SUCCESS {
			klog.Infof("Error shutting down EFML: %v", ret)
		}
	}()

	// FIXME: formalize the full list and document it.
	// http://docs.enflame.com/deploy/xid-errors/index.html#topic_4
	// Application errors: the GCU should still be healthy
	applicationErrorXids := []uint64{
		13, // Graphics Engine Exception
		31, // GCU memory page fault
		43, // GCU stopped processing
		45, // Preemptive cleanup, due to previous errors
		68, // Video processor exception
	}

	skippedXids := make(map[uint64]bool)
	for _, id := range applicationErrorXids {
		skippedXids[id] = true
	}

	for _, additionalXid := range getAdditionalXids(disableHealthChecks) {
		skippedXids[additionalXid] = true
	}

	eventSet, ret := r.efml.EventSetCreate()
	if ret != efml.SUCCESS {
		return fmt.Errorf("failed to create event set: %v", ret)
	}
	defer func() {
		_ = eventSet.Free()
	}()

	parentToDeviceMap := make(map[string]*Device)
	deviceIDToGiMap := make(map[string]int)
	deviceIDToCiMap := make(map[string]int)

	eventMask := uint64(efml.EventTypeXidCriticalError | efml.EventTypeDoubleBitEccError | efml.EventTypeSingleBitEccError)
	for _, d := range devices {
		uuid, gi, ci, err := r.getDevicePlacement(d)
		if err != nil {
			klog.Warningf("Could not determine device placement for %v: %v; Marking it unhealthy.", d.ID, err)
			unhealthy <- d
			continue
		}
		deviceIDToGiMap[d.ID] = gi
		deviceIDToCiMap[d.ID] = ci
		parentToDeviceMap[uuid] = d

		gcu, ret := r.interfaces.GcuDeviceInterfaceGetHandleByUUID(uuid)
		if ret != efml.SUCCESS {
			klog.Infof("unable to get device handle from UUID: %v; marking it as unhealthy", ret)
			unhealthy <- d
			continue
		}

		supportedEvents, ret := gcu.GetSupportedEventTypes()
		if ret != efml.SUCCESS {
			klog.Infof("Unable to determine the supported events for %v: %v; marking it as unhealthy", d.ID, ret)
			unhealthy <- d
			continue
		}

		ret = gcu.RegisterEvents(eventMask&supportedEvents, eventSet)
		if ret == efml.ERROR_NOT_SUPPORTED {
			klog.Warningf("Device %v is too old to support healthchecking.", d.ID)
		}
		if ret != efml.SUCCESS {
			klog.Infof("Marking device %v as unhealthy: %v", d.ID, ret)
			unhealthy <- d
		}
	}

	for {
		select {
		case <-stop:
			return nil
		default:
		}

		e, ret := eventSet.Wait(5000)
		if ret == efml.ERROR_TIMEOUT {
			continue
		}
		if ret != efml.SUCCESS {
			klog.Infof("Error waiting for event: %v; Marking all devices as unhealthy", ret)
			for _, d := range devices {
				unhealthy <- d
			}
			continue
		}

		if e.EventType != efml.EventTypeXidCriticalError {
			klog.Infof("Skipping non-efmlEventTypeXidCriticalError event: %+v", e)
			continue
		}

		if skippedXids[e.EventData] {
			klog.Infof("Skipping event %+v", e)
			continue
		}

		klog.Infof("Processing event %+v", e)
		eventUUID, ret := e.Device.GetUUID()
		if ret != efml.SUCCESS {
			// If we cannot reliably determine the device UUID, we mark all devices as unhealthy.
			klog.Infof("Failed to determine uuid for event %v: %v; Marking all devices as unhealthy.", e, ret)
			for _, d := range devices {
				unhealthy <- d
			}
			continue
		}

		d, exists := parentToDeviceMap[eventUUID]
		if !exists {
			klog.Infof("Ignoring event for unexpected device: %v", eventUUID)
			continue
		}

		if d.IsMigDevice() && e.GcuInstanceId != 0xFFFFFFFF && e.ComputeInstanceId != 0xFFFFFFFF {
			gi := deviceIDToGiMap[d.ID]
			ci := deviceIDToCiMap[d.ID]
			if !(uint32(gi) == e.GcuInstanceId && uint32(ci) == e.ComputeInstanceId) {
				continue
			}
			klog.Infof("Event for mig device %v (gi=%v, ci=%v)", d.ID, gi, ci)
		}

		klog.Infof("XidCriticalError: Xid=%d on Device=%s; marking device as unhealthy.", e.EventData, d.ID)
		unhealthy <- d
	}
}

// getAdditionalXids returns a list of additional Xids to skip from the specified string.
// The input is treaded as a comma-separated string and all valid uint64 values are considered as Xid values. Invalid values
// are ignored.
func getAdditionalXids(input string) []uint64 {
	if input == "" {
		return nil
	}

	var additionalXids []uint64
	for _, additionalXid := range strings.Split(input, ",") {
		trimmed := strings.TrimSpace(additionalXid)
		if trimmed == "" {
			continue
		}
		xid, err := strconv.ParseUint(trimmed, 10, 64)
		if err != nil {
			klog.Infof("Ignoring malformed Xid value %v: %v", trimmed, err)
			continue
		}
		additionalXids = append(additionalXids, xid)
	}

	return additionalXids
}

// getDevicePlacement returns the placement of the specified device.
// For a MIG device the placement is defined by the 3-tuple <parent UUID, GI, CI>
// For a full device the returned 3-tuple is the device's uuid and 0xFFFFFFFF for the other two elements.
func (r *efmlResourceManager) getDevicePlacement(d *Device) (string, int, int, error) {
	if !d.IsMigDevice() {
		return d.GetUUID(), 0xFFFFFFFF, 0xFFFFFFFF, nil
	}
	return r.getMigDeviceParts(d)
}

// getMigDeviceParts returns the parent GI and CI ids of the MIG device.
func (r *efmlResourceManager) getMigDeviceParts(d *Device) (string, int, int, error) {
	if !d.IsMigDevice() {
		return "", 0, 0, fmt.Errorf("cannot get GI and CI of full device")
	}

	uuid := d.GetUUID()
	// For older driver versions, the call to DeviceGetHandleByUUID will fail for MIG devices.
	mig, ret := r.interfaces.GcuDeviceInterfaceGetHandleByUUID(uuid)
	if ret == efml.SUCCESS {
		parentHandle, ret := mig.GetDeviceHandleFromMigDeviceHandle()
		if ret != efml.SUCCESS {
			return "", 0, 0, fmt.Errorf("failed to get parent device handle: %v", ret)
		}

		parentUUID, ret := parentHandle.GetUUID()
		if ret != efml.SUCCESS {
			return "", 0, 0, fmt.Errorf("failed to get parent uuid: %v", ret)
		}
		gi, ret := mig.GetGcuInstanceId()
		if ret != efml.SUCCESS {
			return "", 0, 0, fmt.Errorf("failed to get GCU Instance ID: %v", ret)
		}

		ci, ret := mig.GetComputeInstanceId()
		if ret != efml.SUCCESS {
			return "", 0, 0, fmt.Errorf("failed to get Compute Instance ID: %v", ret)
		}
		return parentUUID, gi, ci, nil
	}
	return parseMigDeviceUUID(uuid)
}

// parseMigDeviceUUID splits the MIG device UUID into the parent device UUID and ci and gi
func parseMigDeviceUUID(mig string) (string, int, int, error) {
	tokens := strings.SplitN(mig, "-", 2)
	if len(tokens) != 2 || tokens[0] != "MIG" {
		return "", 0, 0, fmt.Errorf("Unable to parse UUID as MIG device")
	}

	tokens = strings.SplitN(tokens[1], "/", 3)
	if len(tokens) != 3 || !strings.HasPrefix(tokens[0], "GCU-") {
		return "", 0, 0, fmt.Errorf("Unable to parse UUID as MIG device")
	}

	gi, err := strconv.Atoi(tokens[1])
	if err != nil {
		return "", 0, 0, fmt.Errorf("Unable to parse UUID as MIG device")
	}

	ci, err := strconv.Atoi(tokens[2])
	if err != nil {
		return "", 0, 0, fmt.Errorf("Unable to parse UUID as MIG device")
	}

	return tokens[0], gi, ci, nil
}
*/
