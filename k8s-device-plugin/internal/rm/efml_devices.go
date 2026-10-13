/*
 * Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
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
	"bytes"
	"fmt"
	"os"
	"strconv"

	"enflame/k8s-device-plugin/internal/eflib/interfaces"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/types"
)

const (
	enflameProcDriverPath   = "/proc/driver/enflame"
	enflameCapabilitiesPath = enflameProcDriverPath + "/capabilities"
)

// efmlDevice wraps an interfaces.GcuDeviceInterface with more functions.
type efmlDevice struct {
	interfaces.GcuDeviceInterface
	ID string
}

// efmlMigDevice allows for specific functions of efmlDevice to be overridden.
// type efmlMigDevice efmlDevice

var _ deviceInfo = (*efmlDevice)(nil)

// var _ deviceInfo = (*efmlMigDevice)(nil)

func newGCUDevice(i int, gcu interfaces.GcuDeviceInterface) deviceInfo {
	// index := fmt.Sprintf("%v", i)
	// isWsl, _ := info.New().HasDXCore()
	// if isWsl {
	// 	return index, wslDevice{gcu}
	// }

	return efmlDevice{gcu, fmt.Sprintf("%d", i)}
}

// EF_CLOSE
/*
func newMigDevice(i int, j int, mig interfaces.GcuDeviceInterface) (string, efmlMigDevice) {
	return fmt.Sprintf("%v:%v", i, j), efmlMigDevice{mig}
}

// GetUUID returns the UUID of the device
func (d efmlDevice) GetUUID() string {
	return d.GcuDeviceInterface.GetUUID()
}

// GetUUID returns the UUID of the device
func (d efmlMigDevice) GetUUID() (string, error) {
	return efmlDevice(d).GetUUID()
}

// GetPaths returns the paths for a GCU device
func (d efmlDevice) GetPaths() ([]string, error) {
	minor, err := d.GetMinorNumber()
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s%d", types.GCU_DEV_PATH, minor)

	return []string{path}, nil
}

// GetComputeCapability returns the TOPS Compute Capability for the device.
func (d efmlDevice) GetComputeCapability() (string, error) {
	major, minor, ret := d.Device.GetTopsComputeCapability()
	if ret != efml.SUCCESS {
		return "", ret
	}
	return fmt.Sprintf("%d.%d", major, minor), nil
}

// GetComputeCapability returns the TOPS Compute Capability for the device.
func (d efmlMigDevice) GetComputeCapability() (string, error) {
	parent, ret := d.Device.GetDeviceHandleFromMigDeviceHandle()
	if ret != efml.SUCCESS {
		return "", fmt.Errorf("failed to get parent device: %w", ret)
	}
	return efmlDevice{parent}.GetComputeCapability()
}

// GetPaths returns the paths for a MIG device
func (d efmlMigDevice) GetPaths() ([]string, error) {
	capDevicePaths, err := mig.GetMigCapabilityDevicePaths()
	if err != nil {
		return nil, fmt.Errorf("error getting MIG capability device paths: %v", err)
	}

	gi, ret := d.GetGcuInstanceId()
	if ret != efml.SUCCESS {
		return nil, fmt.Errorf("error getting GCU Instance ID: %v", ret)
	}

	ci, ret := d.GetComputeInstanceId()
	if ret != efml.SUCCESS {
		return nil, fmt.Errorf("error getting Compute Instance ID: %v", ret)
	}

	parent, ret := d.GetDeviceHandleFromMigDeviceHandle()
	if ret != efml.SUCCESS {
		return nil, fmt.Errorf("error getting parent device: %v", ret)
	}
	minor, ret := parent.GetMinorNumber()
	if ret != efml.SUCCESS {
		return nil, fmt.Errorf("error getting GCU device minor number: %v", ret)
	}
	parentPath := fmt.Sprintf("/dev/enflame%d", minor)

	giCapPath := fmt.Sprintf(enflameCapabilitiesPath+"/gcu%d/mig/gi%d/access", minor, gi)
	if _, exists := capDevicePaths[giCapPath]; !exists {
		return nil, fmt.Errorf("missing MIG GCU instance capability path: %v", giCapPath)
	}

	ciCapPath := fmt.Sprintf(enflameCapabilitiesPath+"/gcu%d/mig/gi%d/ci%d/access", minor, gi, ci)
	if _, exists := capDevicePaths[ciCapPath]; !exists {
		return nil, fmt.Errorf("missing MIG GCU instance capability path: %v", giCapPath)
	}

	devicePaths := []string{
		parentPath,
		capDevicePaths[giCapPath],
		capDevicePaths[ciCapPath],
	}

	return devicePaths, nil
}
*/

// GetNumaNode returns the NUMA node associated with the GCU device
func (d efmlDevice) GetNumaNode() (bool, int, error) {
	busId, err := getDeviceBusId(d.GetName())
	if err != nil {
		return false, 0, err
	}
	if busId == "" {
		logger.Warn("device: %s busid not found, numa node scheduler is unabled", d.GetName())
		return false, 0, nil
	}
	numaBytes, err := os.ReadFile(fmt.Sprintf("%s/%s/numa_node", types.DevicePciPrefix, busId))
	if err != nil {
		logger.Error(err, "get gcu device %s numa node failed", d.ID)
		return false, 0, err
	}

	node, err := strconv.Atoi(string(bytes.TrimSpace(numaBytes)))
	if err != nil {
		logger.Error(err, "parsing value for NUMA node: %s failed", string(bytes.TrimSpace(numaBytes)))
		return false, 0, err
	}

	if node < 0 {
		logger.Warn("device: %s numa node: %d is invalid, numa node scheduler is unabled", d.GetName(), node)
		return false, 0, nil
	}

	return true, node, nil
}

// EF_SUPPORT
func getDeviceBusId(deviceName string) (string, error) {
	busIdPath := fmt.Sprintf("%s/%s", types.BusIDPrefix, deviceName)
	content, err := os.ReadFile(busIdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		logger.Error(err, "get gcu device %s busId failed", deviceName)
		return "", err
	}
	return string(bytes.TrimSpace(content)), nil
}

// EF_CLOSE
/*
// GetNumaNode for a MIG device is the NUMA node of the parent device.
func (d efmlMigDevice) GetNumaNode() (bool, int, error) {
	parent, ret := d.GetDeviceHandleFromMigDeviceHandle()
	if ret != efml.SUCCESS {
		return false, 0, fmt.Errorf("error getting parent GCU device from MIG device: %v", ret)
	}

	return efmlDevice{parent}.GetNumaNode()
}

// GetTotalMemory returns the total memory available on the device.
func (d efmlDevice) GetTotalMemory() (uint64, error) {
	info, ret := d.Device.GetMemoryInfo()
	if ret != efml.SUCCESS {
		return 0, ret
	}
	return info.Total, nil
}

// GetTotalMemory returns the total memory available on the device.
func (d efmlMigDevice) GetTotalMemory() (uint64, error) {
	info, ret := d.Device.GetMemoryInfo()
	if ret != efml.SUCCESS {
		return 0, ret
	}
	return info.Total, nil
}
*/
