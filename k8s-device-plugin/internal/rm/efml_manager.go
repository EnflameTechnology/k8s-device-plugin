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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/eflib/interfaces"
	"enflame/k8s-device-plugin/internal/informer"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/types"
)

const (
	//envDisableHealthChecks = "DP_DISABLE_HEALTHCHECKS"
	allHealthChecks = "all"
)

type efmlResourceManager struct {
	resourceManager
	efml map[string]interfaces.HandleInterface
}

var _ ResourceManager = (*efmlResourceManager)(nil)

// NewEFMLResourceManagers returns a set of ResourceManagers, one for each EFML resource in 'config'.
func NewEFMLResourceManagers(deviceHandle map[string]interfaces.HandleInterface, config *spec.Config) ([]ResourceManager, error) {
	// EF_CLOSE
	/*
		ret := efmllib.Init()
		if ret != efml.SUCCESS {
			return nil, fmt.Errorf("failed to initialize EFML: %v", ret)
		}
		defer func() {
			ret := efmllib.Shutdown()
			if ret != efml.SUCCESS {
				klog.Infof("Error shutting down EFML: %v", ret)
			}
		}()
	*/

	deviceMap, err := NewDeviceMap(deviceHandle, config)
	if err != nil {
		return nil, fmt.Errorf("error building device map: %v", err)
	}
	if err := checkDeviceMap(deviceMap, config); err != nil {
		return nil, err
	}

	var rms []ResourceManager
	busidMap, err := findGCUBusID()
	if err != nil {
		return nil, err
	}
	for resourceName, devices := range deviceMap {
		for _, device := range devices {
			busid, ok := busidMap[device.ID]
			if !ok {
				err := fmt.Errorf("pci busID of device: %s not found, busidMap: %v", device.ID, busidMap)
				logger.Error(err)
				return nil, err
			}
			device.PCIBusID = busid
		}
		r := &efmlResourceManager{
			resourceManager: resourceManager{
				config:   config,
				resource: resourceName,
				devices:  devices,
			},
			efml: deviceHandle,
		}
		rms = append(rms, r)
	}

	return rms, nil
}

func findGCUBusID() (map[string]string, error) {
	basePath := types.PCIDevicePath
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logger.Error(err, "read path: %s failed", basePath)
		return nil, err
	}
	result := map[string]string{}
	for _, entry := range entries {
		busID := entry.Name()
		files, err := getGCUPCIDevices(basePath, busID)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			id := strings.Split(file.Name(), "gcu")[1]
			if _, err := strconv.Atoi(id); err != nil {
				continue
			}
			result[id] = busID
			break
		}
	}
	return result, nil
}

// getGCUPCIDevices returns the entries of the driver specific
// subdirectory of a PCI device that exposes gcu<N> nodes. The
// subdirectory is named after the kernel driver, so instead of
// hardcoding driver names the subdirectories are scanned for
// gcu device entries.
func getGCUPCIDevices(basePath, busID string) ([]os.DirEntry, error) {
	deviceDir := filepath.Join(basePath, busID)
	subdirs, err := os.ReadDir(deviceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		logger.Error(err, "read path: %s failed", deviceDir)
		return nil, err
	}
	for _, subdir := range subdirs {
		files, err := os.ReadDir(filepath.Join(deviceDir, subdir.Name()))
		if err != nil {
			continue
		}
		for _, file := range files {
			parts := strings.Split(file.Name(), "gcu")
			if len(parts) != 2 {
				continue
			}
			if _, err := strconv.Atoi(parts[1]); err == nil {
				return files, nil
			}
		}
	}
	return nil, nil
}

func checkDeviceMap(deviceMap DeviceMap, config *spec.Config) error {
	if deviceMap.isEmpty() {
		err := fmt.Errorf("no available device found for resources: %v", config.Resources)
		logger.Error(err)
		return err
	}
	for _, resourceList := range config.Resources {
		for _, resource := range resourceList {
			if len(deviceMap[resource.Name]) == 0 {
				if *config.Flags.FailOnInitError {
					err := fmt.Errorf("resource: %s not found", resource.Name)
					logger.Error(err)
					return err
				}
				logger.Warn("resource: %s not found, skip it", resource.Name)
			}
		}
	}
	return nil
}

// GetPreferredAllocation runs an allocation algorithm over the inputs.
// The algorithm chosen is based both on the incoming set of available devices and various config settings.
func (r *efmlResourceManager) GetPreferredAllocation(available, required []string, size int) ([]string, error) {
	return r.getPreferredAllocation(available, required, size)
}

// GetDevicePaths returns the required and optional device nodes for the requested resources
func (r *efmlResourceManager) GetDevicePaths(ids []string) []string {
	paths := []string{
		"/dev/gcuctl",
		// "/dev/enflame-uvm",
		// "/dev/enflame-uvm-tools",
		// "/dev/enflame-modeset",
	}

	return append(paths, r.Devices().Subset(ids).GetPaths()...)
}

// CheckHealth performs health checks on a set of devices, writing to the 'reportList' channel with any healthy status changed devices
func (r *efmlResourceManager) CheckHealth(stop <-chan interface{}, reportList chan<- []*Device, podInformer *informer.PodInformer) {
	logger.Info("Device health check work start")
	watchHealth(stop, reportList, r.devices, podInformer)
}

// getPreferredAllocation runs an allocation algorithm over the inputs.
// The algorithm chosen is based both on the incoming set of available devices and various config settings.
func (r *efmlResourceManager) getPreferredAllocation(available, required []string, size int) ([]string, error) {
	// If all of the available devices are full GCUs without replicas, then
	// calculate an aligned allocation across those devices.
	if r.Devices().AlignedAllocationSupported() && !AnnotatedIDs(available).AnyHasAnnotations() {
		return r.alignedAlloc(available, required, size)
	}

	// Otherwise, distribute them evenly across all replicated GCUs
	return r.distributedAlloc(available, required, size)
}

// alignedAlloc shells out to the alignedAllocationPolicy that is set in
// order to calculate the preferred allocation.
func (r *efmlResourceManager) alignedAlloc(available, required []string, size int) ([]string, error) {
	// EF_CLOSE
	/*
		var devices []string

		linkedDevices, err := gcuallocator.NewDevices(
			gcuallocator.WithEfmlLib(r.efml),
		)
		if err != nil {
			return nil, fmt.Errorf("unable to get device link information: %w", err)
		}

		availableDevices, err := linkedDevices.Filter(available)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve list of available devices: %v", err)
		}

		requiredDevices, err := linkedDevices.Filter(required)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve list of required devices: %v", err)
		}

		allocatedDevices := gcuallocator.NewBestEffortPolicy().Allocate(availableDevices, requiredDevices, size)
		for _, device := range allocatedDevices {
			devices = append(devices, device.UUID)
		}

		return devices, nil
	*/
	return []string{}, nil
}
