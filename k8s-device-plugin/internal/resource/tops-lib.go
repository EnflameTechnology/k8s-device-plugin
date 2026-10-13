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

type topsLib struct{}

var _ Manager = (*topsLib)(nil)

// NewTopsManager returns an resource manger for TOPS devices
func NewTopsManager() Manager {
	return &topsLib{}
}

// GetDevices returns the TOPS devices available on the system
func (l *topsLib) GetDevices() ([]Device, error) {
	count, r := tops.DeviceGetCount()
	if r != tops.SUCCESS {
		return nil, fmt.Errorf("failed to get number of TOPS devices: %v", r)
	}

	var devices []Device
	for i := 0; i < count; i++ {
		d, r := tops.DeviceGet(i)
		if r != tops.SUCCESS {
			return nil, fmt.Errorf("failed to get TOPS device %v: %v", i, r)
		}
		devices = append(devices, NewTopsDevice(d))
	}

	return devices, nil
}

// GetTopsDriverVersion returns the TOPS driver version
func (l *topsLib) GetTopsDriverVersion() (*uint, *uint, error) {
	version, r := tops.DriverGetVersion()
	if r != tops.SUCCESS {
		return nil, nil, fmt.Errorf("failed to get driver version: %v", r)
	}

	major := uint(version) / 1000
	minor := uint(version) % 100 / 10

	return &major, &minor, nil
}

// GetDriverVersion returns the driver version.
// This is currently "unknown" for Tegra systems.
func (l *topsLib) GetDriverVersion() (string, error) {
	return "unknown.unknown.unknown", nil
}

// Init initializes the TOPS library.
func (l *topsLib) Init() error {
	r := tops.Init()
	if r != tops.SUCCESS {
		return fmt.Errorf("%v", r)
	}
	return nil
}

// Shutdown shuts down the TOPS library.
func (l *topsLib) Shutdown() (err error) {
	r := tops.Shutdown()
	if r != tops.SUCCESS {
		return fmt.Errorf("%v", r)
	}
	return nil
}
*/