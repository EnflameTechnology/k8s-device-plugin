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
	// "github.com/ENFLAME/go-eflib/pkg/eflib/info"
	"k8s.io/klog/v2"

	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/cdi"
	"enflame/k8s-device-plugin/internal/eflib/interfaces"
)

type manager struct {
	migStrategy     string
	failOnInitError bool
	// EF_CLOSE
	// efmllib         efml.Interface
	deviceHandle map[string]interfaces.HandleInterface

	cdiHandler map[spec.ResourceName]cdi.CDIInterface
	config     *spec.Config
	// EF_CLOSE
	// infolib    info.Interface
}

// New creates a new plugin manager with the supplied options.
func New(opts ...Option) (Interface, error) {
	m := &manager{}
	for _, opt := range opts {
		opt(m)
	}

	if m.config == nil {
		klog.Warning("no config provided, returning a null manager")
		return &null{}, nil
	}

	// if m.cdiHandler == nil {
	// 	m.cdiHandler = cdi.NewNullHandler()
	// }
	// EF_CLOSE
	/*
		if m.infolib == nil {
			m.infolib = info.New()
		}
	*/
	// EF_SUPPORT
	return (*efmlmanager)(m), nil
	// EF_CLOSE
	/*
		mode, err := m.resolveMode()
		if err != nil {
			return nil, err
		}

		switch mode {
		case "efml":
			if m.efmllib == nil {
				m.efmllib = efml.New()
			}
			ret := m.efmllib.Init()
			if ret != efml.SUCCESS {
				klog.Errorf("Failed to initialize EFML: %v.", ret)
				klog.Errorf("If this is a GCU node, did you set the docker default runtime to `enflame`?")
				klog.Errorf("You can check the prerequisites at: https://enflame/k8s-device-plugin#prerequisites")
				klog.Errorf("You can learn how to set the runtime at: https://enflame/k8s-device-plugin#quick-start")
				klog.Errorf("If this is not a GCU node, you should set up a toleration or nodeSelector to only deploy this plugin on GCU nodes")
				if m.failOnInitError {
					return nil, fmt.Errorf("efml init failed: %v", ret)
				}
				klog.Warningf("efml init failed: %v", ret)
				return &null{}, nil
			}
			defer func() {
				_ = m.efmllib.Shutdown()
			}()

			return (*efmlmanager)(m), nil
		case "tegra":
			return (*tegramanager)(m), nil
		case "null":
			return &null{}, nil
		}

		return nil, fmt.Errorf("unknown mode: %v", mode)
	*/
}

// EF_CLOSE
/*
func (m *manager) resolveMode() (string, error) {
	// logWithReason logs the output of the has* / is* checks from the info.Interface
	logWithReason := func(f func() (bool, string), tag string) bool {
		is, reason := f()
		if !is {
			tag = "non-" + tag
		}
		klog.Infof("Detected %v platform: %v", tag, reason)
		return is
	}

	hasEFML := logWithReason(m.infolib.HasEfml, "EFML")
	isTegra := logWithReason(m.infolib.IsTegraSystem, "Tegra")

	if !hasEFML && !isTegra {
		klog.Error("Incompatible platform detected")
		klog.Error("If this is a GCU node, did you configure the ENFLAME Container Toolkit?")
		klog.Error("You can check the prerequisites at: https://github.com/ENFLAME/k8s-device-plugin#prerequisites")
		klog.Error("You can learn how to set the runtime at: https://github.com/ENFLAME/k8s-device-plugin#quick-start")
		klog.Error("If this is not a GCU node, you should set up a toleration or nodeSelector to only deploy this plugin on GCU nodes")
		if m.failOnInitError {
			return "", fmt.Errorf("platform detection failed")
		}
		return "null", nil
	}

	// The ENFLAME container stack does not yet support the use of integrated AND discrete GCUs on the same node.
	if isTegra {
		if hasEFML {
			klog.Warning("Disabling Tegra-based resources on EFML system")
			return "efml", nil
		}
		return "tegra", nil
	}

	return "efml", nil
}
*/
