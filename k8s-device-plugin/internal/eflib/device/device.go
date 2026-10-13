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
	"fmt"

	"enflame/k8s-device-plugin/internal/eflib/interfaces"
)

// Device defines the set of extended functions associated with a device.Device
type Device interface {
	interfaces.GcuDeviceInterface
	// GetArchitectureAsString() (string, error)
	// GetBrandAsString() (string, error)
	// GetTopsComputeCapabilityAsString() (string, error)
	// GetMigDevices() ([]MigDevice, error)
	// GetMigProfiles() ([]MigProfile, error)
	IsMigCapable() (bool, error)
	IsMigEnabled() (bool, error)
	// VisitMigDevices(func(j int, m MigDevice) error) error
	// VisitMigProfiles(func(p MigProfile) error) error
}

type device struct {
	interfaces.GcuDeviceInterface
	lib *devicelib
	// migProfiles []MigProfile
}

var _ Device = &device{}

// NewDevice builds a new Device from an interfaces.GcuDeviceInterface
func (d *devicelib) NewDevice(dev interfaces.GcuDeviceInterface) (Device, error) {
	return d.newDevice(dev)
}

// NewDeviceByUUID builds a new Device from a UUID
// func (d *devicelib) NewDeviceByUUID(uuid string) (Device, error) {
// 	id := strings.Split(uuid, "-")
// 	index, err := strconv.Atoi(id[1])
// 	if err != nil {
// 		logger.Error(err, "get device index from uuid: %s failed", uuid)
// 		return nil, err
// 	}
// 	dev := d.efml.DeviceGetHandleByIndex(index)
// 	return d.newDevice(dev)
// }

// newDevice creates a device from an interfaces.GcuDeviceInterface
func (d *devicelib) newDevice(dev interfaces.GcuDeviceInterface) (*device, error) {
	return &device{dev, d}, nil
}

// GetArchitectureAsString returns the Device architecture as a string
// func (d *device) GetArchitectureAsString() (string, error) {
// 	arch, ret := d.GetArchitecture()
// 	if ret != efml.SUCCESS {
// 		return "", fmt.Errorf("error getting device architecture: %v", ret)
// 	}
// 	switch arch {
// 	case interfaces.GcuDeviceInterface_ARCH_KEPLER:
// 		return "Kepler", nil
// 	case interfaces.GcuDeviceInterface_ARCH_MAXWELL:
// 		return "Maxwell", nil
// 	case interfaces.GcuDeviceInterface_ARCH_PASCAL:
// 		return "Pascal", nil
// 	case interfaces.GcuDeviceInterface_ARCH_VOLTA:
// 		return "Volta", nil
// 	case interfaces.GcuDeviceInterface_ARCH_TURING:
// 		return "Turing", nil
// 	case interfaces.GcuDeviceInterface_ARCH_AMPERE:
// 		return "Ampere", nil
// 	case interfaces.GcuDeviceInterface_ARCH_ADA:
// 		return "Ada", nil
// 	case interfaces.GcuDeviceInterface_ARCH_HOPPER:
// 		return "Hopper", nil
// 	case interfaces.GcuDeviceInterface_ARCH_UNKNOWN:
// 		return "Unknown", nil
// 	}
// 	return "", fmt.Errorf("error interpreting device architecture as string: %v", arch)
// }

// GetBrandAsString returns the Device architecture as a string
// func (d *device) GetBrandAsString() (string, error) {
// 	brand, ret := d.GetBrand()
// 	if ret != efml.SUCCESS {
// 		return "", fmt.Errorf("error getting device brand: %v", ret)
// 	}
// 	switch brand {
// 	case efml.BRAND_UNKNOWN:
// 		return "Unknown", nil
// 	case efml.BRAND_QUADRO:
// 		return "Quadro", nil
// 	case efml.BRAND_TESLA:
// 		return "Tesla", nil
// 	case efml.BRAND_EFS:
// 		return "EFS", nil
// 	case efml.BRAND_GRID:
// 		return "Grid", nil
// 	case efml.BRAND_GEFORCE:
// 		return "GeForce", nil
// 	case efml.BRAND_TITAN:
// 		return "Titan", nil
// 	case efml.BRAND_ENFLAME_VAPPS:
// 		return "EnflameVApps", nil
// 	case efml.BRAND_ENFLAME_VPC:
// 		return "EnflameVPC", nil
// 	case efml.BRAND_ENFLAME_VCS:
// 		return "EnflameVCS", nil
// 	case efml.BRAND_ENFLAME_VWS:
// 		return "EnflameVWS", nil
// 	// Deprecated in favor of efml.BRAND_ENFLAME_CLOUD_GAMING
// 	//case efml.BRAND_ENFLAME_VGAMING:
// 	//	return "VGaming", nil
// 	case efml.BRAND_ENFLAME_CLOUD_GAMING:
// 		return "EnflameCloudGaming", nil
// 	case efml.BRAND_QUADRO_RTX:
// 		return "QuadroRTX", nil
// 	case efml.BRAND_ENFLAME_RTX:
// 		return "EnflameRTX", nil
// 	case efml.BRAND_ENFLAME:
// 		return "Enflame", nil
// 	case efml.BRAND_GEFORCE_RTX:
// 		return "GeForceRTX", nil
// 	case efml.BRAND_TITAN_RTX:
// 		return "TitanRTX", nil
// 	}
// 	return "", fmt.Errorf("error interpreting device brand as string: %v", brand)
// }

// GetTopsComputeCapabilityAsString returns the Device's TOPS compute capability as a version string
// func (d *device) GetTopsComputeCapabilityAsString() (string, error) {
// 	major, minor, ret := d.GetTopsComputeCapability()
// 	if ret != efml.SUCCESS {
// 		return "", fmt.Errorf("error getting TOPS compute capability: %v", ret)
// 	}
// 	return fmt.Sprintf("%d.%d", major, minor), nil
// }

// IsMigCapable checks if a device is capable of having MIG paprtitions created on it
func (d *device) IsMigCapable() (bool, error) {
	// if !d.lib.hasSymbol("efmlDeviceGetMigMode") {
	// 	return false, nil
	// }

	// _, _, ret := interfaces.GcuDeviceInterface(d).GetMigMode()
	// if ret == efml.ERROR_NOT_SUPPORTED {
	// 	return false, nil
	// }
	// if ret != efml.SUCCESS {
	// 	return false, fmt.Errorf("error getting MIG mode: %v", ret)
	// }

	// return true, nil
	return false, nil
}

// IsMigEnabled checks if a device has MIG mode currently enabled on it
func (d *device) IsMigEnabled() (bool, error) {
	// if !d.lib.hasSymbol("efmlDeviceGetMigMode") {
	// 	return false, nil
	// }

	// mode, _, ret := interfaces.GcuDeviceInterface(d).GetMigMode()
	// if ret == efml.ERROR_NOT_SUPPORTED {
	// 	return false, nil
	// }
	// if ret != efml.SUCCESS {
	// 	return false, fmt.Errorf("error getting MIG mode: %v", ret)
	// }

	// return (mode == interfaces.GcuDeviceInterface_MIG_ENABLE), nil
	return false, nil
}

// VisitMigDevices walks a top-level device and invokes a callback function for each MIG device configured on it
// func (d *device) VisitMigDevices(visit func(int, MigDevice) error) error {
// 	capable, err := d.IsMigCapable()
// 	if err != nil {
// 		return fmt.Errorf("error checking if GCU is MIG capable: %v", err)
// 	}
// 	if !capable {
// 		return nil
// 	}

// 	count, ret := interfaces.GcuDeviceInterface(d).GetMaxMigDeviceCount()
// 	if ret != efml.SUCCESS {
// 		return fmt.Errorf("error getting max MIG device count: %v", ret)
// 	}

// 	for i := 0; i < count; i++ {
// 		device, ret := interfaces.GcuDeviceInterface(d).GetMigDeviceHandleByIndex(i)
// 		if ret == efml.ERROR_NOT_FOUND {
// 			continue
// 		}
// 		if ret == efml.ERROR_INVALID_ARGUMENT {
// 			continue
// 		}
// 		if ret != efml.SUCCESS {
// 			return fmt.Errorf("error getting MIG device handle at index '%v': %v", i, ret)
// 		}
// 		mig, err := d.lib.NewMigDevice(device)
// 		if err != nil {
// 			return fmt.Errorf("error creating new MIG device wrapper: %v", err)
// 		}
// 		err = visit(i, mig)
// 		if err != nil {
// 			return fmt.Errorf("error visiting MIG device: %v", err)
// 		}
// 	}
// 	return nil
// }

// VisitMigProfiles walks a top-level device and invokes a callback function for each unique MIG Profile that can be configured on it
// func (d *device) VisitMigProfiles(visit func(MigProfile) error) error {
// 	capable, err := d.IsMigCapable()
// 	if err != nil {
// 		return fmt.Errorf("error checking if GCU is MIG capable: %v", err)
// 	}

// 	if !capable {
// 		return nil
// 	}

// 	memory, ret := d.GetMemoryInfo()
// 	if ret != efml.SUCCESS {
// 		return fmt.Errorf("error getting device memory info: %v", ret)
// 	}

// 	for i := 0; i < efml.GCU_INSTANCE_PROFILE_COUNT; i++ {
// 		giProfileInfo, ret := d.GetGcuInstanceProfileInfo(i)
// 		if ret == efml.ERROR_NOT_SUPPORTED {
// 			continue
// 		}
// 		if ret == efml.ERROR_INVALID_ARGUMENT {
// 			continue
// 		}
// 		if ret != efml.SUCCESS {
// 			return fmt.Errorf("error getting GCU Instance profile info: %v", ret)
// 		}

// 		for j := 0; j < efml.COMPUTE_INSTANCE_PROFILE_COUNT; j++ {
// 			for k := 0; k < efml.COMPUTE_INSTANCE_ENGINE_PROFILE_COUNT; k++ {
// 				p, err := d.lib.NewMigProfile(i, j, k, giProfileInfo.MemorySizeMB, memory.Total)
// 				if err != nil {
// 					return fmt.Errorf("error creating MIG profile: %v", err)
// 				}

// 				// NOTE: The EFML API doesn't currently let us query the set of
// 				// valid Compute Instance profiles without first instantiating
// 				// a GCU Instance to check against. In theory, it should be
// 				// possible to get this information without a reference to a
// 				// GCU instance, but no API is provided for that at the moment.
// 				// We run the checks below to weed out invalid profiles
// 				// heuristically, given what we know about how they are
// 				// physically constructed. In the future we should do this via
// 				// EFML once a proper API for this exists.
// 				pi := p.GetInfo()
// 				if pi.C > pi.G {
// 					continue
// 				}
// 				if (pi.C < pi.G) && ((pi.C * 2) > (pi.G + 1)) {
// 					continue
// 				}

// 				err = visit(p)
// 				if err != nil {
// 					return fmt.Errorf("error visiting MIG profile: %v", err)
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }

// // GetMigDevices gets the set of MIG devices associated with a top-level device
// func (d *device) GetMigDevices() ([]MigDevice, error) {
// 	var migs []MigDevice
// 	err := d.VisitMigDevices(func(j int, m MigDevice) error {
// 		migs = append(migs, m)
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return migs, nil
// }

// // GetMigProfiles gets the set of unique MIG profiles associated with a top-level device
// func (d *device) GetMigProfiles() ([]MigProfile, error) {
// 	// Return the cached list if available
// 	if d.migProfiles != nil {
// 		return d.migProfiles, nil
// 	}

// 	// Otherwise generate it...
// 	var profiles []MigProfile
// 	err := d.VisitMigProfiles(func(p MigProfile) error {
// 		profiles = append(profiles, p)
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	// And cache it before returning
// 	d.migProfiles = profiles
// 	return profiles, nil
// }

// isSkipped checks whether the device should be skipped.
func (d *device) isSkipped() (bool, error) {
	name := d.GetName()
	if _, exists := d.lib.skippedDevices[name]; exists {
		return true, nil
	}

	return false, nil
}

// VisitDevices visits each top-level device and invokes a callback function for it
func (d *devicelib) VisitDevices(visit func(int, Device) error) error {
	for _, deviceHandle := range d.efml {
		devIDPathMap := deviceHandle.GetDeviceIDPathMap()
		for devID := range devIDPathMap {
			gcuDevice := deviceHandle.DeviceGetHandleByIndex(devID)
			dev := &device{gcuDevice, d}
			isSkipped, err := dev.isSkipped()
			if err != nil {
				return fmt.Errorf("error checking whether device is skipped: %v", err)
			}
			if isSkipped {
				continue
			}

			err = visit(devID, dev)
			if err != nil {
				return fmt.Errorf("error visiting device: %v", err)
			}
		}
	}
	return nil
}

// VisitMigDevices walks a top-level device and invokes a callback function for each MIG device configured on it
// func (d *devicelib) VisitMigDevices(visit func(int, Device, int, MigDevice) error) error {
// 	err := d.VisitDevices(func(i int, dev Device) error {
// 		err := dev.VisitMigDevices(func(j int, mig MigDevice) error {
// 			err := visit(i, dev, j, mig)
// 			if err != nil {
// 				return fmt.Errorf("error visiting MIG device: %v", err)
// 			}
// 			return nil
// 		})
// 		if err != nil {
// 			return fmt.Errorf("error visiting device: %v", err)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return fmt.Errorf("error visiting devices: %v", err)
// 	}
// 	return nil
// }

// // VisitMigProfiles walks a top-level device and invokes a callback function for each unique MIG profile found on them
// func (d *devicelib) VisitMigProfiles(visit func(MigProfile) error) error {
// 	visited := make(map[string]bool)
// 	err := d.VisitDevices(func(i int, dev Device) error {
// 		err := dev.VisitMigProfiles(func(p MigProfile) error {
// 			if visited[p.String()] {
// 				return nil
// 			}

// 			err := visit(p)
// 			if err != nil {
// 				return fmt.Errorf("error visiting MIG profile: %v", err)
// 			}

// 			visited[p.String()] = true
// 			return nil
// 		})
// 		if err != nil {
// 			return fmt.Errorf("error visiting device: %v", err)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return fmt.Errorf("error visiting devices: %v", err)
// 	}
// 	return nil
// }

// GetDevices gets the set of all top-level devices
func (d *devicelib) GetDevices() ([]Device, error) {
	var devs []Device
	// err := d.VisitDevices(func(i int, dev Device) error {
	// 	devs = append(devs, dev)
	// 	return nil
	// })
	// if err != nil {
	// 	return nil, err
	// }
	return devs, nil
}

// GetMigDevices gets the set of MIG devices across all top-level devices
// func (d *devicelib) GetMigDevices() ([]MigDevice, error) {
// 	var migs []MigDevice
// 	err := d.VisitMigDevices(func(i int, dev Device, j int, m MigDevice) error {
// 		migs = append(migs, m)
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return migs, nil
// }

// // GetMigProfiles gets the set of unique MIG profiles across all top-level devices
// func (d *devicelib) GetMigProfiles() ([]MigProfile, error) {
// 	// Return the cached list if available
// 	if d.migProfiles != nil {
// 		return d.migProfiles, nil
// 	}

// 	// Otherwise generate it...
// 	var profiles []MigProfile
// 	err := d.VisitMigProfiles(func(p MigProfile) error {
// 		profiles = append(profiles, p)
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	// And cache it before returning
// 	d.migProfiles = profiles
// 	return profiles, nil
// }

// // hasSymbol checks to see if the given symbol is present in the EFML library.
// // If devicelib is configured to not verify symbols, then all symbols are assumed to exist.
// func (d *devicelib) hasSymbol(symbol string) bool {
// 	if !*d.verifySymbols {
// 		return true
// 	}

// 	return d.efml.Lookup(symbol) == nil
// }
