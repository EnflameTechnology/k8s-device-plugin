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

	// "github.com/ENFLAME/go-eflib/pkg/eflib/device"
	// "github.com/ENFLAME/go-eflib/pkg/efml"
)

type efmlDevice struct {
	device.Device
	devicelib device.Interface
}

var _ Device = (*efmlDevice)(nil)

// GetMigDevices returns the list of MIG devices configured on this device
func (d efmlDevice) GetMigDevices() ([]Device, error) {
	migs, err := d.Device.GetMigDevices()
	if err != nil {
		return nil, err
	}

	var devices []Device
	for _, m := range migs {
		device := efmlMigDevice{
			MigDevice: m,
			devicelib: d.devicelib,
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// GetTopsComputeCapability returns the TOPS major and minor versions.
func (d efmlDevice) GetTopsComputeCapability() (int, int, error) {
	major, minor, ret := d.Device.GetTopsComputeCapability()
	if ret != efml.SUCCESS {
		return 0, 0, ret
	}

	return major, minor, nil
}

// GetAttributes is only supported for MIG devices.
func (d efmlDevice) GetAttributes() (map[string]interface{}, error) {
	return nil, fmt.Errorf("GetAttributes is not supported for non-MIG devices")
}

// GetDeviceHandleFromMigDeviceHandle is only supported for MIG devices
func (d efmlDevice) GetDeviceHandleFromMigDeviceHandle() (Device, error) {
	return nil, fmt.Errorf("GetDeviceHandleFromMigDeviceHandle is not supported for non-MIG devices")
}

// GetName returns the device name / model.
func (d efmlDevice) GetName() (string, error) {
	name, ret := d.Device.GetName()
	if ret != efml.SUCCESS {
		return "", ret
	}
	return name, nil
}

// GetTotalMemoryMB returns the total memory on a device in MB
func (d efmlDevice) GetTotalMemoryMB() (uint64, error) {
	info, ret := d.Device.GetMemoryInfo()
	if ret != efml.SUCCESS {
		return 0, ret
	}
	return info.Total / (1024 * 1024), nil
}
*/
