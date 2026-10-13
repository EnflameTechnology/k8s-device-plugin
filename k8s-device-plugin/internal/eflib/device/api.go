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

package device

import (
	v1 "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/eflib/gcu"
	"enflame/k8s-device-plugin/internal/eflib/interfaces"
)

// Interface provides the API to the 'device' package
type DeviceInterface interface {
	GetDevices() ([]Device, error)
	VisitDevices(func(i int, d Device) error) error
	NewDevice(d interfaces.GcuDeviceInterface) (Device, error)
	// NewDeviceByUUID(uuid string) (Device, error)
	// GetMigDevices() ([]MigDevice, error)
	// GetMigProfiles() ([]MigProfile, error)
	// NewMigDevice(d interfaces.GcuDeviceInterface) (MigDevice, error)
	// NewMigDeviceByUUID(uuid string) (MigDevice, error)
	// NewMigProfile(giProfileID, ciProfileID, ciEngProfileID int, migMemorySizeMB, deviceMemorySizeBytes uint64) (MigProfile, error)
	// ParseMigProfile(profile string) (MigProfile, error)
	// VisitMigDevices(func(i int, d Device, j int, m MigDevice) error) error
	// VisitMigProfiles(func(p MigProfile) error) error
	// AssertValidMigProfileFormat(profile string) error
}

type devicelib struct {
	efml           map[string]interfaces.HandleInterface
	skippedDevices map[string]struct{}
	verifySymbols  *bool
	// migProfiles    []MigProfile
}

var _ DeviceInterface = &devicelib{}

// New creates a new instance of the 'device' interface
func New(config *v1.Config, opts ...Option) DeviceInterface {
	d := &devicelib{}
	for _, opt := range opts {
		opt(d)
	}
	if d.efml == nil {
		d.efml = gcu.NewRegisterDeviceLib(config)
	}
	if d.verifySymbols == nil {
		verify := true
		d.verifySymbols = &verify
	}
	if d.skippedDevices == nil {
		WithSkippedDevices(
			"DGX Display",
			"ENFLAME DGX Display",
		)(d)
	}
	return d
}

// WithEfml provides an Option to set the EFML library used by the 'device' interface
func WithEfml(efml map[string]interfaces.HandleInterface) Option {
	return func(d *devicelib) {
		d.efml = efml
	}
}

// WithVerifySymbols provides an option to toggle whether to verify select symbols exist in dynamic libraries before calling them
func WithVerifySymbols(verify bool) Option {
	return func(d *devicelib) {
		d.verifySymbols = &verify
	}
}

// WithSkippedDevices provides an Option to set devices to be skipped by model name
func WithSkippedDevices(names ...string) Option {
	return func(d *devicelib) {
		if d.skippedDevices == nil {
			d.skippedDevices = make(map[string]struct{})
		}
		for _, name := range names {
			d.skippedDevices[name] = struct{}{}
		}
	}
}

// Option defines a function for passing options to the New() call
type Option func(*devicelib)
