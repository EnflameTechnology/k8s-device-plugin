/**
# Copyright (c) 2021-2022, NVIDIA CORPORATION.  All rights reserved.
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

package vgcu

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// MockVGCU represents mock of VGCU interface
type MockVGCU struct {
	devices []*Device
}

// Devices returns VGCU devices with mocked data
func (p *MockVGCU) Devices() ([]*Device, error) {
	return p.devices, nil
}

func TestIsVGCUDevice(t *testing.T) {
	mockVGCU := NewMockVGCU().(*Lib)
	devices, _ := mockVGCU.pci.Devices()
	for _, device := range devices {
		// check for vendor id
		require.Equal(t, "0x10de", fmt.Sprintf("0x%x", GetWord(device.Config, 0)), "Enflame PCI Vendor ID")
		// check for vendor capability records
		capability, err := device.GetVendorSpecificCapability()
		require.NoError(t, err, "Get vendor capabilities from configuration space")
		require.NotZero(t, len(capability), "Vendor capability record")
		if device.Address == "passthrough" {
			require.False(t, mockVGCU.IsVGCUDevice(capability), "Is not a virtual GCU device")
			require.Equal(t, 20, len(capability), "Vendor capability length for passthrough device")
		}
		if device.Address == "vgcu" {
			require.Equal(t, 27, len(capability), "Vendor capability length for vgcu device")
			require.Equal(t, uint8(9), GetByte(capability, 0), "Vendor capability ID")
		}
	}
}

func TestVGCUGetInfo(t *testing.T) {
	devices, _ := NewMockVGCU().Devices()
	for _, device := range devices {
		if device.pci.Address == "vgcu" {
			require.NotEmpty(t, device.pci.Config, "Device Configuration data")
			require.Equal(t, len(device.pci.Config), 256, "Device configuration data length")

			require.NotEmpty(t, device.vGCUCapability, "Vendor capability record")
			require.Equal(t, device.vGCUCapability[0], uint8(9), "Vendor capability id")

			info, err := device.GetInfo()
			require.NoError(t, err, "Get host driver version and branch")
			require.NotNil(t, info, "Host driver info")
			require.Equal(t, "460.16", info.HostDriverVersion, "Host driver version")
			require.Equal(t, "r460_00", info.HostDriverBranch, "Host driver branch")
		}
	}
}
