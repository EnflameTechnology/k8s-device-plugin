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

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	. "github.com/agiledragon/gomonkey"
	. "github.com/smartystreets/goconvey/convey"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	v1 "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/eflib/interfaces"
	"enflame/k8s-device-plugin/internal/rm"
	"enflame/k8s-device-plugin/internal/tests"
	rmmock "enflame/k8s-device-plugin/internal/tests/rm"
	"enflame/k8s-device-plugin/internal/types"
)

func TestNewEnflameDevicePlugin(t *testing.T) {
	Convey("test NewEnflameDevicePlugin\n", t, func() {
		configFile := `{
			"version": "v1",
			"flags": {
				"migStrategy": "none",
				"failOnInitError": true,
				"mpsRoot": "",
				"enflameDriverRoot": "/",
				"gdsEnabled": false,
				"mofedEnabled": false,
				"useNodeFeatureAPI": null,
				"plugin": {
					"passDeviceSpecs": false,
					"pcieSwitchAffinity": false,
					"deviceListStrategy": [
						"envvar"
					],
					"deviceIDStrategy": "index",
					"cdiAnnotationPrefix": "cdi.k8s.io/",
					"enflameCTKPath": "/usr/bin/enflame-ctk",
					"containerDriverRoot": "/driver-root"
				}
			}
		}
		`
		config := &v1.Config{}
		err := json.Unmarshal([]byte(configFile), config)
		So(err, ShouldEqual, nil)
		Convey("1. test NewEnflameDevicePlugin success\n", func() {
			plugin, err := NewEnflameDevicePlugin(config, &rmmock.ResourceManager{ResourceName: "enflame.com/gcu"}, nil, nil)
			So(err, ShouldEqual, nil)
			So(plugin.config, ShouldEqual, config)
		})

		Convey("2. test NewEnflameDevicePlugin use pcie switch affinity\n", func() {
			*config.Flags.Plugin.PcieSwitchAffinity = true
			patcheNewPcieSwitchAffinity := ApplyFunc(NewPcieSwitchAffinity, func(_ interfaces.HandleInterface, _ string) (*PcieSwitchAffinity, error) {
				return &PcieSwitchAffinity{}, nil
			})
			defer patcheNewPcieSwitchAffinity.Reset()
			plugin, err := NewEnflameDevicePlugin(config, &rmmock.ResourceManager{ResourceName: "enflame.com/gcu"}, nil, nil)
			So(err, ShouldEqual, nil)
			So(plugin.config, ShouldEqual, config)
			So(plugin.switchAffinity.eflib, ShouldEqual, nil)
			So(len(plugin.switchAffinity.switchTopoList), ShouldEqual, 0)
			So(plugin.switchAffinity.podResource, ShouldEqual, nil)
			So(*plugin.config.Flags.Plugin.PcieSwitchAffinity, ShouldEqual, true)
		})
	})
}

func TestListAndWatch(t *testing.T) {
	Convey("test ListAndWatch\n", t, func() {
		Convey("1. test ListAndWatch failed\n", func() {
			plugin := &EnflameDevicePlugin{rm: &rmmock.ResourceManager{}}
			errMock := fmt.Errorf("send error")
			patcheSend := ApplyMethod(reflect.TypeOf(new(tests.ListAndWatchServer)), "Send", func(_ *tests.ListAndWatchServer,
				_ *pluginapi.ListAndWatchResponse) error {
				return errMock
			})
			defer patcheSend.Reset()
			err := plugin.ListAndWatch(&pluginapi.Empty{}, &tests.ListAndWatchServer{})
			So(err, ShouldEqual, errMock)
		})

		Convey("2. test ListAndWatch stop\n", func() {
			plugin := &EnflameDevicePlugin{rm: &rmmock.ResourceManager{}, stop: make(chan interface{})}
			patcheSend := ApplyMethod(reflect.TypeOf(new(tests.ListAndWatchServer)), "Send", func(_ *tests.ListAndWatchServer,
				_ *pluginapi.ListAndWatchResponse) error {
				return nil
			})
			defer patcheSend.Reset()
			go func() {
				plugin.stop <- true
			}()
			err := plugin.ListAndWatch(&pluginapi.Empty{}, &tests.ListAndWatchServer{})
			So(err, ShouldEqual, nil)
		})

		Convey("3. test ListAndWatch device unhealth and send error\n", func() {
			plugin := &EnflameDevicePlugin{rm: &rmmock.ResourceManager{}, health: make(chan []*rm.Device)}
			errMock := fmt.Errorf("send error")
			outputsSend := []OutputCell{
				{Values: Params{nil}},
				{Values: Params{errMock}},
			}
			patcheSend := ApplyMethodSeq(reflect.TypeOf(new(tests.ListAndWatchServer)), "Send", outputsSend)
			defer patcheSend.Reset()

			go func() {
				plugin.health <- []*rm.Device{{Device: pluginapi.Device{Health: pluginapi.Unhealthy}}}
			}()
			err := plugin.ListAndWatch(&pluginapi.Empty{}, &tests.ListAndWatchServer{})
			So(err, ShouldEqual, errMock)
		})

		Convey("4. test ListAndWatch device health and stop success\n", func() {
			plugin := &EnflameDevicePlugin{rm: &rmmock.ResourceManager{}, health: make(chan []*rm.Device), stop: make(chan interface{})}
			patcheSend := ApplyMethod(reflect.TypeOf(new(tests.ListAndWatchServer)), "Send", func(_ *tests.ListAndWatchServer,
				_ *pluginapi.ListAndWatchResponse) error {
				return nil
			})
			defer patcheSend.Reset()

			go func() {
				plugin.health <- []*rm.Device{{Device: pluginapi.Device{Health: pluginapi.Healthy}}}
				plugin.stop <- true
			}()
			err := plugin.ListAndWatch(&pluginapi.Empty{}, &tests.ListAndWatchServer{})
			So(err, ShouldEqual, nil)
		})
	})
}

func TestAllocate(t *testing.T) {
	Convey("test Allocate\n", t, func() {
		boolFalse, boolTrue, uuid, root := false, true, types.DeviceIDStrategyUUID, "/"
		Convey("1. test Allocate success\n", func() {
			plugin := &EnflameDevicePlugin{
				rm: &rmmock.ResourceManager{},
				config: &v1.Config{
					Flags: v1.Flags{
						CommandLineFlags: v1.CommandLineFlags{
							Plugin: &v1.PluginCommandLineFlags{
								PcieSwitchAffinity: &boolFalse,
								DeviceIDStrategy:   &uuid,
								PassDeviceSpecs:    &boolTrue,
							},
							EnflameDriverRoot: &root,
							GDSEnabled:        &boolFalse,
							MOFEDEnabled:      &boolFalse,
						},
					},
				},
				deviceListStrategies: v1.DeviceListStrategies{v1.DeviceListStrategyEnvvar: true},
				deviceListEnvvar:     types.ENFLAME_VISIBLE_DEVICES,
			}
			reqs := &pluginapi.AllocateRequest{
				ContainerRequests: []*pluginapi.ContainerAllocateRequest{
					{
						DevicesIDs: []string{"0", "1"},
					},
				},
			}
			patcheGetDevicePaths := ApplyMethod(reflect.TypeOf(new(rmmock.ResourceManager)), "GetDevicePaths", func(_ *rmmock.ResourceManager,
				_ []string) []string {
				return []string{"/dev/gcu0", "/dev/gcu1"}
			})
			defer patcheGetDevicePaths.Reset()
			res, err := plugin.Allocate(context.Background(), reqs)
			So(err, ShouldEqual, nil)
			So(len(res.ContainerResponses), ShouldEqual, 1)
			deviceIDs, ok := res.ContainerResponses[0].Envs[types.ENFLAME_VISIBLE_DEVICES]
			So(ok, ShouldEqual, true)
			So(deviceIDs, ShouldEqual, "0,1")
			So(len(res.ContainerResponses[0].Devices), ShouldEqual, 2)
			So(res.ContainerResponses[0].Devices[0].ContainerPath, ShouldEqual, "/dev/gcu0")
			So(res.ContainerResponses[0].Devices[0].HostPath, ShouldEqual, "/dev/gcu0")
			So(res.ContainerResponses[0].Devices[0].Permissions, ShouldEqual, "rw")
			So(res.ContainerResponses[0].Devices[1].ContainerPath, ShouldEqual, "/dev/gcu1")
			So(res.ContainerResponses[0].Devices[1].HostPath, ShouldEqual, "/dev/gcu1")
			So(res.ContainerResponses[0].Devices[1].Permissions, ShouldEqual, "rw")
		})
	})
}
