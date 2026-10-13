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
	"github.com/ENFLAME/go-eflib/pkg/eflib/device"
	"github.com/ENFLAME/go-eflib/pkg/efml"
)

type efmlLib struct {
	efml.Interface
	devicelib device.Interface
}

// NewEFMLManager creates a new manager that uses EFML to query and manage devices
func NewEFMLManager() Manager {
	efmllib := efml.New()
	devicelib := device.New(device.WithEfml(efmllib))

	m := efmlLib{
		Interface: efmllib,
		devicelib: devicelib,
	}
	return m
}

// GetTopsDriverVersion : Return the tops v using EFML
func (l efmlLib) GetTopsDriverVersion() (*uint, *uint, error) {
	v, ret := l.Interface.SystemGetTopsDriverVersion()
	if ret != efml.SUCCESS {
		return nil, nil, ret
	}
	major := uint(v / 1000)
	minor := uint(v % 1000 / 10)

	return &major, &minor, nil
}

// GetDevices returns the EFML devices for the manager
func (l efmlLib) GetDevices() ([]Device, error) {
	libdevices, err := l.devicelib.GetDevices()
	if err != nil {
		return nil, err
	}

	var devices []Device
	for _, d := range libdevices {
		device := efmlDevice{
			Device:    d,
			devicelib: l.devicelib,
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// GetDriverVersion returns the driver version
func (l efmlLib) GetDriverVersion() (string, error) {
	v, ret := l.Interface.SystemGetDriverVersion()
	if ret != efml.SUCCESS {
		return "", ret
	}
	return v, nil
}

// Init initialises the library
func (l efmlLib) Init() error {
	ret := l.Interface.Init()
	if ret != efml.SUCCESS {
		return ret
	}
	return nil
}

// Shutdown shuts down the library
func (l efmlLib) Shutdown() error {
	ret := l.Interface.Shutdown()
	if ret != efml.SUCCESS {
		return ret
	}
	return nil
}
*/
