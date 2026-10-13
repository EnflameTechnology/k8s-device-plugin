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
	"strings"
)

// Interface allows us to get a list of vGCU specific PCI devices
type Interface interface {
	Devices() ([]*Device, error)
}

// Device is just an alias to a PCIDevice
type Device struct {
	pci            *PCIDevice
	vGCUCapability []byte
}

// Info represents vGCU driver info running on underlying hypervisor host.
type Info struct {
	HostDriverVersion string
	HostDriverBranch  string
}

const (
	// VGCUCapabilityRecordStart indicates offset of beginning vGCU capability record
	VGCUCapabilityRecordStart = 5
	// HostDriverVersionLength indicates max length of driver version
	HostDriverVersionLength = 10
	// HostDriverBranchLength indicates max length of driver branch
	HostDriverBranchLength = 10
)

// Lib implements the EnflameVGCU interface
type Lib struct {
	pci EnflamePCI
}

// NewVGCULib returns an instance of Lib implementing the VGCU interface
func NewVGCULib(pci EnflamePCI) Interface {
	return &Lib{pci: pci}
}

// NewMockVGCU initializes and returns mock Interface interface type
func NewMockVGCU() Interface {
	return NewVGCULib(NewMockEnflamePCI())
}

// Devices returns all vGCU devices attached to the guest
func (v *Lib) Devices() ([]*Device, error) {
	pciDevices, err := v.pci.Devices()
	if err != nil {
		return nil, fmt.Errorf("error getting ENFLAME specific PCI devices: %v", err)
	}

	var vgcus []*Device
	for _, device := range pciDevices {
		capability, err := device.GetVendorSpecificCapability()
		if err != nil {
			return nil, fmt.Errorf("unable to read vendor specific capability for %s: %v", device.Address, err)
		}
		if capability == nil {
			continue
		}
		if exists := v.IsVGCUDevice(capability); exists {
			vgcu := &Device{
				pci:            device,
				vGCUCapability: capability,
			}
			vgcus = append(vgcus, vgcu)
		}
	}
	return vgcus, nil
}

// IsVGCUDevice returns true if the device is of type vGCU
func (v *Lib) IsVGCUDevice(capability []byte) bool {
	if len(capability) < 5 {
		return false
	}
	// check for vGCU signature, 0x56, 0x46 i.e "VF"
	if capability[3] != 0x56 {
		return false
	}
	if capability[4] != 0x46 {
		return false
	}
	return true
}

// GetInfo returns information about vGCU manager running on the underlying hypervisor host
func (d *Device) GetInfo() (*Info, error) {
	if len(d.vGCUCapability) == 0 {
		return nil, fmt.Errorf("vendor capability record is not populated for device %s", d.pci.Address)
	}

	// traverse vGCU vendor capability records until host driver version record(id: 0) is found
	var hostDriverVersion string
	var hostDriverBranch string
	foundDriverVersionRecord := false
	pos := VGCUCapabilityRecordStart
	record := GetByte(d.vGCUCapability, VGCUCapabilityRecordStart)
	for record != 0 && pos < len(d.vGCUCapability) {
		// find next record
		recordLength := GetByte(d.vGCUCapability, pos+1)
		pos += int(recordLength)
		record = GetByte(d.vGCUCapability, pos)
	}

	if record == 0 && pos+2+HostDriverVersionLength+HostDriverBranchLength <= len(d.vGCUCapability) {
		foundDriverVersionRecord = true
		// found vGCU host driver version record type
		// initialized at record data byte, i.e pos + 1(record id byte) + 1(record lengh byte)
		i := pos + 2
		// 10 bytes of driver version
		for ; i < pos+2+HostDriverVersionLength; i++ {
			hostDriverVersion += string(GetByte(d.vGCUCapability, i))
		}
		hostDriverVersion = strings.Trim(hostDriverVersion, "\x00")
		// 10 bytes of driver branch
		for ; i < pos+2+HostDriverVersionLength+HostDriverBranchLength; i++ {
			hostDriverBranch += string(GetByte(d.vGCUCapability, i))
		}
		hostDriverBranch = strings.Trim(hostDriverBranch, "\x00")
	}

	if !foundDriverVersionRecord {
		return nil, fmt.Errorf("cannot find driver version record in vendor specific capability for device %s", d.pci.Address)
	}

	info := &Info{
		HostDriverVersion: hostDriverVersion,
		HostDriverBranch:  hostDriverBranch,
	}

	return info, nil
}
