/*
 * Copyright (c) 2019-2022, NVIDIA CORPORATION.  All rights reserved.
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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY Type, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rm

import (
	"errors"
	"fmt"

	// EF_CLOSE
	/*
		"strings"
		"github.com/ENFLAME/go-eflib/pkg/eflib/device"
		"github.com/ENFLAME/go-eflib/pkg/eflib/info"
		"github.com/ENFLAME/go-eflib/pkg/efml"
		"k8s.io/klog/v2"
	*/
	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/logger"
	"enflame/k8s-device-plugin/internal/informer"
)

// resourceManager forms the base type for specific resource manager implementations
type resourceManager struct {
	config   *spec.Config
	resource spec.ResourceName
	devices  Devices
}

// ResourceManager provides an interface for listing a set of Devices and checking health on them
type ResourceManager interface {
	Resource() spec.ResourceName
	Devices() Devices
	GetDevicePaths([]string) []string
	GetPreferredAllocation(available, required []string, size int) ([]string, error)
	CheckHealth(stop <-chan interface{}, unhealthy chan<- []*Device, podInformer *informer.PodInformer)
	ValidateRequest(AnnotatedIDs) error
}

// EF_CLOSE
/*
// NewResourceManagers returns a []ResourceManager, one for each resource in 'config'.
func NewResourceManagers(efmllib efml.Interface, config *spec.Config) ([]ResourceManager, error) {
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

	if !hasEFML && !isTegra {
		klog.Error("Incompatible platform detected")
		klog.Error("If this is a GCU node, did you configure the ENFLAME Container Toolkit?")
		klog.Error("You can check the prerequisites at: https://enflame/k8s-device-plugin#prerequisites")
		klog.Error("You can learn how to set the runtime at: https://enflame/k8s-device-plugin#quick-start")
		klog.Error("If this is not a GCU node, you should set up a toleration or nodeSelector to only deploy this plugin on GCU nodes")
		if *config.Flags.FailOnInitError {
			return nil, fmt.Errorf("platform detection failed")
		}
		return nil, nil
	}

	// The ENFLAME container stack does not yet support the use of integrated AND discrete GCUs on the same node.
	if hasEFML && isTegra {
		klog.Warning("Disabling Tegra-based resources on EFML system")
		isTegra = false
	}

	var resourceManagers []ResourceManager

	if hasEFML {
		efmlManagers, err := NewEFMLResourceManagers(efmllib, config)
		if err != nil {
			return nil, fmt.Errorf("failed to construct EFML resource managers: %v", err)
		}
		resourceManagers = append(resourceManagers, efmlManagers...)
	}

	if isTegra {
		tegraManagers, err := NewTegraResourceManagers(config)
		if err != nil {
			return nil, fmt.Errorf("failed to construct Tegra resource managers: %v", err)
		}
		resourceManagers = append(resourceManagers, tegraManagers...)
	}

	return resourceManagers, nil
}
*/

// Resource gets the resource name associated with the ResourceManager
func (r *resourceManager) Resource() spec.ResourceName {
	return r.resource
}

// Resource gets the devices managed by the ResourceManager
func (r *resourceManager) Devices() Devices {
	return r.devices
}

var errInvalidRequest = errors.New("invalid request")

// ValidateRequests checks the requested IDs against the resource manager configuration.
// It asserts that all requested IDs are known to the resource manager and that the request is
// valid for a specified sharing configuration.
func (r *resourceManager) ValidateRequest(ids AnnotatedIDs) error {
	// Assert that all requested IDs are known to the resource manager
	for _, id := range ids {
		if !r.devices.Contains(id) {
			err := fmt.Errorf("%w: unknown device: %s", errInvalidRequest, id)
			logger.Error(err)
			return err
		}
	}

	// If the devices being allocated are replicas, then (conditionally)
	// error out if more than one resource is being allocated.
	includesReplicas := ids.AnyHasAnnotations()
	numRequestedDevices := len(ids)
	switch r.config.Sharing.SharingStrategy() {
	case spec.SharingStrategyTimeSlicing:
		if includesReplicas && numRequestedDevices > 1 && r.config.Sharing.ReplicatedResources().FailRequestsGreaterThanOne {
			err := fmt.Errorf("%w: maximum request size for shared resources is 1; found %d", errInvalidRequest, numRequestedDevices)
			logger.Error(err)
			return err
		}
	case spec.SharingStrategyMPS:
		// For MPS sharing, we explicitly ignore the FailRequestsGreaterThanOne
		// value in the sharing settings.
		// This setting was added to timeslicing after the initial release and
		// is set to `false` to maintain backward compatibility with existing
		// deployments. If we do extend MPS to allow multiple devices to be
		// requested, the MPS API will be extended separately from the
		// time-slicing API.
		if includesReplicas && numRequestedDevices > 1 {
			return fmt.Errorf("%w: maximum request size for shared resources is 1; found %d", errInvalidRequest, numRequestedDevices)
		}
	}
	return nil
}

// AddDefaultResourcesToConfig adds default resource matching rules to config.Resources
func AddDefaultResourcesToConfig(config *spec.Config) error {
	for _, resourceName := range *config.Flags.Plugin.CustomRegisterResource {
		if err := config.Resources.AddResource("*", spec.ResourceName(resourceName)); err != nil {
			return err
		}
	}

	if config.Flags.MigStrategy == nil {
		return nil
	}
	// EF_CLOSE
	/*
		switch *config.Flags.MigStrategy {
		case spec.MigStrategySingle:
			return config.Resources.AddMIGResource("*", "gcu")
		case spec.MigStrategyMixed:
			hasEFML, reason := info.New().HasEfml()
			if !hasEFML {
				klog.Warningf("mig-strategy=%q is only supported with EFML", spec.MigStrategyMixed)
				klog.Warningf("EFML not detected: %v", reason)
				return nil
			}

			efmllib := efml.New()
			ret := efmllib.Init()
			if ret != efml.SUCCESS {
				if *config.Flags.FailOnInitError {
					return fmt.Errorf("failed to initialize EFML: %v", ret)
				}
				return nil
			}
			defer func() {
				ret := efmllib.Shutdown()
				if ret != efml.SUCCESS {
					klog.Errorf("Error shutting down EFML: %v", ret)
				}
			}()

			devicelib := device.New(
				device.WithEfml(efmllib),
			)
			return devicelib.VisitMigProfiles(func(p device.MigProfile) error {
				info := p.GetInfo()
				if info.C != info.G {
					return nil
				}
				resourceName := strings.ReplaceAll("mig-"+p.String(), "+", ".")
				return config.Resources.AddMIGResource(p.String(), resourceName)
			})
		}
	*/
	return nil
}
