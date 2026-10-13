/*
 * Copyright (c) 2020, NVIDIA CORPORATION.  All rights reserved.
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

package main

import (
	"fmt"
	// "github.com/ENFLAME/go-eflib/pkg/efml"

	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/cdi"
	"enflame/k8s-device-plugin/internal/eflib/gcu"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/plugin/manager"
)

// NewPluginManager creates an EFML-based plugin manager
func NewPluginManager(config *spec.Config) (manager.Interface, bool, error) {
	var err error
	switch *config.Flags.MigStrategy {
	case spec.MigStrategyNone:
	case spec.MigStrategySingle:
	case spec.MigStrategyMixed:
	default:
		return nil, false, fmt.Errorf("unknown strategy: %v", *config.Flags.MigStrategy)
	}
	// EF_CLOSE
	/*
		efmllib := efml.New()

		deviceListStrategies, err := spec.NewDeviceListStrategies(*config.Flags.Plugin.DeviceListStrategy)
		if err != nil {
			return nil, fmt.Errorf("invalid device list strategy: %v", err)
		}

		cdiHandler, err := cdi.New(
			cdi.WithDeviceListStrategies(deviceListStrategies),
			cdi.WithDriverRoot(*config.Flags.Plugin.ContainerDriverRoot),
			cdi.WithTargetDriverRoot(*config.Flags.EnflameDriverRoot),
			cdi.WithEnflameCTKPath(*config.Flags.Plugin.EnflameCTKPath),
			cdi.WithEfml(efmllib),
			cdi.WithDeviceIDStrategy(*config.Flags.Plugin.DeviceIDStrategy),
			cdi.WithVendor("k8s.device-plugin.enflame.com"),
			cdi.WithGdsEnabled(*config.Flags.GDSEnabled),
			cdi.WithMofedEnabled(*config.Flags.MOFEDEnabled),
		)
		if err != nil {
			return nil, false, fmt.Errorf("unable to create cdi handler: %v", err)
		}
	*/
	// EF_SUPPORT
	// Reserve the interface for subsequent expansion
	// deviceHandle := gcu.NewRegisterDeviceLib(config)
	// cdiHandler, _ := cdi.NewHandler("k8s.device-plugin.enflame.com")
	deviceHandle := gcu.NewRegisterDeviceLib(config)
	cdiEnabled := false
	for _, listStrategy := range *config.Flags.Plugin.DeviceListStrategy {
		if listStrategy == "cdi-annotations" {
			logger.Info("cdi-annotations is enabled")
			cdiEnabled = true
		}
	}
	cdiHandler, err := cdi.NewHandler(config, deviceHandle, cdiEnabled)
	if err != nil {
		return nil, false, err
	}

	m, err := manager.New(
		// manager.WithEFML(efmllib),
		manager.WithEFML(deviceHandle),
		manager.WithCDIHandler(cdiHandler),
		manager.WithConfig(config),
		manager.WithFailOnInitError(*config.Flags.FailOnInitError),
		manager.WithMigStrategy(*config.Flags.MigStrategy),
	)
	if err != nil {
		return nil, false, fmt.Errorf("unable to create plugin manager: %v", err)
	}

	if cdiHandler != nil {
		if err := m.CreateCDISpecFile(); err != nil {
			return nil, false, fmt.Errorf("unable to create cdi spec file: %v", err)
		}
	}

	return m, cdiEnabled, nil
}
