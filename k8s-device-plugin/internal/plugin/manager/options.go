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
	// "github.com/ENFLAME/go-eflib/pkg/efml"
	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/cdi"
	"enflame/k8s-device-plugin/internal/eflib/interfaces"
)

// Option is a function that configures a manager
type Option func(*manager)

// WithCDIHandler sets the CDI handler for the manager
func WithCDIHandler(handler map[spec.ResourceName]cdi.CDIInterface) Option {
	return func(m *manager) {
		m.cdiHandler = handler
	}
}

// WithEFML sets the EFML handler for the manager
/*
func WithEFML(efmllib efml.Interface) Option {
	return func(m *manager) {
		m.efmllib = efmllib
	}
}
*/
// EF_SUPPORT
func WithEFML(deviceHandle map[string]interfaces.HandleInterface) Option {
	return func(m *manager) {
		m.deviceHandle = deviceHandle
	}
}

// WithFailOnInitError sets whether the manager should fail on initialization errors
func WithFailOnInitError(failOnInitError bool) Option {
	return func(m *manager) {
		m.failOnInitError = failOnInitError
	}
}

// WithMigStrategy sets the MIG strategy for the manager
func WithMigStrategy(migStrategy string) Option {
	return func(m *manager) {
		m.migStrategy = migStrategy
	}
}

// WithConfig sets the config reference for the manager
func WithConfig(config *spec.Config) Option {
	return func(m *manager) {
		m.config = config
	}
}
