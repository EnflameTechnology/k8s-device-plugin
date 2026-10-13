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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package interfaces

// Interface defines the functions implemented by an EFML library
//
//go:generate moq -out efml_mock.go . Interface
type HandleInterface interface {
	// DeviceGetCount() int
	GetDeviceIDPathMap() map[int]string
	DeviceGetHandleByIndex(Index int) GcuDeviceInterface
	// DeviceGetHandleByUUID(UUID string) (Device, Return)
	// ErrorString(r Return) string
	// EventSetCreate() (EventSet, Return)
	// Init() Return
	// Lookup(string) error
	// Shutdown() Return
	// SystemGetTopsDriverVersion() (int, Return)
	// SystemGetDriverVersion() (string, Return)
}

// Device defines the functions implemented by an EFML device
//
//go:generate moq -out device_mock.go . Device
type GcuDeviceInterface interface {
	// CreateGcuInstanceWithPlacement(*GcuInstanceProfileInfo, *GcuInstancePlacement) (GcuInstance, Return)
	// GetArchitecture() (DeviceArchitecture, Return)
	// GetAttributes() (DeviceAttributes, Return)
	// GetBrand() (BrandType, Return)
	// GetComputeInstanceId() (int, Return)
	// GetTopsComputeCapability() (int, int, Return)
	// GetDeviceHandleFromMigDeviceHandle() (Device, Return)
	// GetGcuInstanceById(ID int) (GcuInstance, Return)
	// GetGcuInstanceId() (int, Return)
	// GetGcuInstancePossiblePlacements(*GcuInstanceProfileInfo) ([]GcuInstancePlacement, Return)
	// GetGcuInstanceProfileInfo(Profile int) (GcuInstanceProfileInfo, Return)
	// GetGcuInstances(Info *GcuInstanceProfileInfo) ([]GcuInstance, Return)
	// GetIndex() (int, Return)
	// GetMaxMigDeviceCount() (int, Return)
	// GetMemoryInfo() (Memory, Return)
	// GetMigDeviceHandleByIndex(Index int) (Device, Return)
	// GetMigMode() (int, int, Return)
	GetMinorNumber() (int, error)
	GetName() string
	DeviceType() string
	// GetNvLinkRemotePciInfo(int) (PciInfo, Return)
	// GetNvLinkState(int) (EnableState, Return)
	// GetPciInfo() (PciInfo, Return)
	// GetSupportedEventTypes() (uint64, Return)
	// GetTopologyCommonAncestor(Device) (GcuTopologyLevel, Return)
	GetUUID() string
	GetPaths() ([]string, error)
	// IsVgcu() bool
	// IsMigDeviceHandle() (bool, Return)
	// RegisterEvents(uint64, EventSet) Return
	// SetComputeMode(ComputeMode) Return
	// SetMigMode(Mode int) (Return, Return)
	// // efmlDeviceHandle returns a pointer to the underlying EFML device.
	// efmlDeviceHandle() *efml.Device
}
