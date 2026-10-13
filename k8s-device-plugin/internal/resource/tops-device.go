/**
# Copyright (c) 2022, NVIDIA CORPORATION.  All rights reserved.
# Copyright (c) 2024, ENFLAME INC.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package resource
// EF_CLOSE
/*
import (
	"fmt"

	"enflame/k8s-device-plugin/internal/tops"
)

type topsDevice tops.Device

var _ Device = (*topsDevice)(nil)

// NewTopsDevice constructs a new TOPS device
func NewTopsDevice(d tops.Device) Device {
	device := topsDevice(d)
	return &device
}

// GetAttributes is unsupported for TOPS devices
func (d *topsDevice) GetAttributes() (map[string]interface{}, error) {
	return nil, fmt.Errorf("GetAttributes is not supported for TOPS devices")
}

// GetTopsComputeCapability returns the TOPS Compute Capability major and minor versions.
// If the device is a MIG device (i.e. a compute instance) these are 0
func (d *topsDevice) GetTopsComputeCapability() (int, int, error) {
	major, r := tops.Device(*d).GetAttribute(tops.COMPUTE_CAPABILITY_MAJOR)
	if r != tops.SUCCESS {
		return 0, 0, fmt.Errorf("failed to get TOPS compute capability major for device: result=%v", r)
	}

	minor, r := tops.Device(*d).GetAttribute(tops.COMPUTE_CAPABILITY_MINOR)
	if r != tops.SUCCESS {
		return 0, 0, fmt.Errorf("failed to get TOPS compute capability minor for device: result=%v", r)
	}

	return major, minor, nil
}

// GetDeviceHandleFromMigDeviceHandle is unsupported for TOPS devices
func (d *topsDevice) GetDeviceHandleFromMigDeviceHandle() (Device, error) {
	return nil, fmt.Errorf("GetDeviceHandleFromMigDeviceHandle is unsupported for TOPS devices")
}

// GetTotalMemoryMB returns the total memory for a device
func (d *topsDevice) GetTotalMemoryMB() (uint64, error) {
	total, r := tops.Device(*d).TotalMem()
	if r != tops.SUCCESS {
		return 0, fmt.Errorf("failed to get memory info for device: %v", r)
	}
	return total / (1024 * 1024), nil
}

// GetMigDevices is unsupported for TOPS devices
func (d *topsDevice) GetMigDevices() ([]Device, error) {
	return nil, fmt.Errorf("GetMigDevices is unsupported for TOPS devices")
}

// GetName returns the device name / model.
func (d *topsDevice) GetName() (string, error) {
	name, r := tops.Device(*d).GetName()
	if r != tops.SUCCESS {
		return "", fmt.Errorf("failed to get device name: %v", r)
	}

	return name, nil
}

// GetUUID is unsupported for TOPS devices
func (d *topsDevice) GetUUID() (string, error) {
	return "", fmt.Errorf("GetUUID is unsupported for TOPS devices")
}

// IsMigCapable always returns false for TOPS devices
func (d *topsDevice) IsMigCapable() (bool, error) {
	return false, nil
}

// IsMigEnabled always returns false for TOPS devices
func (d *topsDevice) IsMigEnabled() (bool, error) {
	return false, nil
}
*/