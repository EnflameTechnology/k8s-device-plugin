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

import (
	// "github.com/ENFLAME/go-eflib/pkg/eflib/info"
	"k8s.io/klog/v2"

	spec "enflame/k8s-device-plugin/api/config/v1"
)

// NewManager is a factory method that creates a resource Manager based on the specified config.
func NewManager(config *spec.Config) Manager {
	return WithConfig(getManager(), config)
}

// WithConfig modifies a manager depending on the specified config.
// If failure on a call to init is allowed, the manager is wrapped to allow fallback to a Null manager.
func WithConfig(manager Manager, config *spec.Config) Manager {
	if *config.Flags.FailOnInitError {
		return manager
	}

	return NewFallbackToNullOnInitError(manager)
}

// getManager returns the resource manager depending on the system configuration.
func getManager() Manager {
// EF_CLOSE
/*
	// logWithReason logs the output of the has* / is* checks from the info.Interface
	logWithReason := func(f func() (bool, string), tag string) bool {
		is, reason := f()
		if !is {
			tag = "non-" + tag
		}
		klog.Infof("Detected %v platform: %v", tag, reason)
		return is
	}

	infolib := info.New()

	hasEFML := logWithReason(infolib.HasEfml, "EFML")
	isTegra := logWithReason(infolib.IsTegraSystem, "Tegra")

	// The ENFLAME container stack does not yet support the use of integrated AND discrete GCUs on the same node.
	if hasEFML && isTegra {
		klog.Warning("Disabling Tegra-based resources on EFML system")
		isTegra = false
	}

	if hasEFML {
		klog.Info("Using EFML manager")
		return NewEFMLManager()
	} else if isTegra {
		klog.Info("Using TOPS manager")
		return NewTopsManager()
	}
*/
	klog.Warning("No valid resources detected; using empty manager.")
	return NewNullManager()
}
