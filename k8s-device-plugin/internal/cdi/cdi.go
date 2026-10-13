/*
 * Copyright (c) 2023, NVIDIA CORPORATION.  All rights reserved.
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

package cdi

// EF_CLOSE
/*
import (
	"fmt"
	"path/filepath"

	efdevice "github.com/ENFLAME/go-eflib/pkg/eflib/device"
	"github.com/ENFLAME/go-eflib/pkg/efml"
	"enflame/enflame-container-toolkit/pkg/efcdi"
	transformroot "enflame/enflame-container-toolkit/pkg/efcdi/transform/root"
	"github.com/sirupsen/logrus"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"
	cdiparser "tags.cncf.io/container-device-interface/pkg/parser"

	spec "enflame/k8s-device-plugin/api/config/v1"
)

const (
	cdiRoot = "/var/run/cdi"
)

// cdiHandler creates CDI specs for devices assocatied with the device plugin
type cdiHandler struct {
	logger           *logrus.Logger
	efml             efml.Interface
	efdevice         efdevice.Interface
	driverRoot       string
	targetDriverRoot string
	enflameCTKPath    string
	vendor           string
	deviceIDStrategy string

	deviceListStrategies spec.DeviceListStrategies

	gdsEnabled   bool
	mofedEnabled bool

	cdilibs map[string]efcdi.Interface
}

var _ Interface = &cdiHandler{}

// newHandler constructs a new instance of the 'cdi' interface
func newHandler(opts ...Option) (Interface, error) {
	c := &cdiHandler{}
	for _, opt := range opts {
		opt(c)
	}

	if !c.deviceListStrategies.IsCDIEnabled() {
		return &null{}, nil
	}

	if c.logger == nil {
		c.logger = logrus.StandardLogger()
	}
	if c.efml == nil {
		c.efml = efml.New()
	}
	if c.efdevice == nil {
		c.efdevice = efdevice.New(efdevice.WithEfml(c.efml))
	}
	if c.deviceIDStrategy == "" {
		c.deviceIDStrategy = "uuid"
	}
	if c.driverRoot == "" {
		c.driverRoot = "/"
	}
	if c.targetDriverRoot == "" {
		c.targetDriverRoot = c.driverRoot
	}

	deviceNamer, err := efcdi.NewDeviceNamer(c.deviceIDStrategy)
	if err != nil {
		return nil, err
	}

	c.cdilibs = make(map[string]efcdi.Interface)

	c.cdilibs["gcu"], err = efcdi.New(
		efcdi.WithLogger(c.logger),
		efcdi.WithEfmlLib(c.efml),
		efcdi.WithDeviceLib(c.efdevice),
		efcdi.WithENFLAMECTKPath(c.enflameCTKPath),
		efcdi.WithDriverRoot(c.driverRoot),
		efcdi.WithDeviceNamers(deviceNamer),
		efcdi.WithVendor(c.vendor),
		efcdi.WithClass("gcu"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create efcdi library: %v", err)
	}

	var additionalModes []string
	if c.gdsEnabled {
		additionalModes = append(additionalModes, "gds")
	}
	if c.mofedEnabled {
		additionalModes = append(additionalModes, "mofed")
	}

	for _, mode := range additionalModes {
		lib, err := efcdi.New(
			efcdi.WithLogger(c.logger),
			efcdi.WithENFLAMECTKPath(c.enflameCTKPath),
			efcdi.WithDriverRoot(c.driverRoot),
			efcdi.WithVendor(c.vendor),
			efcdi.WithMode(mode),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create efcdi library: %v", err)
		}
		c.cdilibs[mode] = lib
	}

	return c, nil
}

// CreateSpecFile creates a CDI spec file for the specified devices.
func (cdi *cdiHandler) CreateSpecFile() error {
	for class, cdilib := range cdi.cdilibs {
		cdi.logger.Infof("Generating CDI spec for resource: %s/%s", cdi.vendor, class)

		if class == "gcu" {
			ret := cdi.efml.Init()
			if ret != efml.SUCCESS {
				return fmt.Errorf("failed to initialize EFML: %v", ret)
			}
			defer func() {
				_ = cdi.efml.Shutdown()
			}()
		}

		spec, err := cdilib.GetSpec()
		if err != nil {
			return fmt.Errorf("failed to get CDI spec: %v", err)
		}

		err = transformroot.New(
			transformroot.WithRoot(cdi.driverRoot),
			transformroot.WithTargetRoot(cdi.targetDriverRoot),
			transformroot.WithRelativeTo("host"),
		).Transform(spec.Raw())
		if err != nil {
			return fmt.Errorf("failed to transform driver root in CDI spec: %v", err)
		}

		specName, err := cdiapi.GenerateNameForSpec(spec.Raw())
		if err != nil {
			return fmt.Errorf("failed to generate spec name: %v", err)
		}

		err = spec.Save(filepath.Join(cdiRoot, specName+".json"))
		if err != nil {
			return fmt.Errorf("failed to save CDI spec: %v", err)
		}
	}

	return nil
}

// QualifiedName constructs a CDI qualified device name for the specified resources.
// Note: This assumes that the specified id matches the device name returned by the naming strategy.
func (cdi *cdiHandler) QualifiedName(class string, id string) string {
	return cdiparser.QualifiedName(cdi.vendor, class, id)
}
*/