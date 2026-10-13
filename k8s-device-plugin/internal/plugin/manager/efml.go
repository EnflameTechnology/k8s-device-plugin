/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
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

package manager

import (
	"fmt"

	"enflame/k8s-device-plugin/internal/plugin"
	"enflame/k8s-device-plugin/internal/rm"

	"enflame/k8s-device-plugin/internal/logger"
)

type efmlmanager manager

// GetPlugins returns the plugins associated with the EFML resources available on the node
func (m *efmlmanager) GetPlugins() ([]plugin.Interface, error) {
	logger.Info("get enflame plugins start")
	// EF_CLOSE
	// rms, err := rm.NewNVMLResourceManagers(m.efmllib, m.config)
	rms, err := rm.NewEFMLResourceManagers(m.deviceHandle, m.config)
	if err != nil {
		return nil, fmt.Errorf("failed to construct EFML resource managers: %v", err)
	}

	var plugins []plugin.Interface
	for _, r := range rms {
		// EF_CLOSE
		// plugin, err := plugin.NewEnflameDevicePlugin(m.config, r, m.cdiHandler)
		plugin, err := plugin.NewEnflameDevicePlugin(m.config, r, m.cdiHandler[r.Resource()], m.deviceHandle[r.Resource().ResourceClass()])
		if err != nil {
			return nil, fmt.Errorf("failed to create plugin: %w", err)
		}
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}

// CreateCDISpecFile creates forwards the request to the CDI handler
func (m *efmlmanager) CreateCDISpecFile() error {
	// EF_CLOSE
	for _, cdiHandler := range m.cdiHandler {
		if err := cdiHandler.CreateSpecFile(); err != nil {
			return err
		}
	}
	return nil
	// return nil
}
