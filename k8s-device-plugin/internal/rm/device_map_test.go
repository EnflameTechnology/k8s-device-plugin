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

package rm

import (
	"encoding/json"
	"testing"

	// . "github.com/agiledragon/gomonkey"
	. "github.com/smartystreets/goconvey/convey"

	v1 "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/eflib/gcu"
	"enflame/k8s-device-plugin/internal/tests"
)

func TestNewDeviceMap(t *testing.T) {
	Convey("test NewDeviceMap\n", t, func() {
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
		config.Resources = v1.Resources{
			"gcu": []v1.Resource{
				{
					Pattern: "*",
					Name:    "enflame.com/gcu",
				},
				{
					Pattern: "*",
					Name:    "enflame-tech.com/gcu",
				},
			},
			"vgcu": []v1.Resource{
				{
					Pattern: "*",
					Name:    "enflame.com/vgcu",
				},
			},
		}

		Convey("1. test NewDeviceMap gcu device success\n", func() {
			fakeDeviceCount, deviceType := 8, "gcu"
			fakeDeviceList, err := tests.CreateFakeDevice(deviceType, fakeDeviceCount)
			defer func() {
				So(tests.ClearFakeDevice(fakeDeviceList), ShouldEqual, nil)
			}()
			So(err, ShouldEqual, nil)
			res, err := NewDeviceMap(gcu.NewRegisterDeviceLib(config), config)
			So(err, ShouldEqual, nil)
			So(res.isEmpty(), ShouldEqual, false)
			So(len(res), ShouldEqual, 2)
			devices, ok := res["enflame.com/gcu"]
			So(ok, ShouldEqual, true)
			So(len(devices), ShouldEqual, 8)
			devices, ok = res["enflame-tech.com/gcu"]
			So(ok, ShouldEqual, true)
			So(len(devices), ShouldEqual, 8)
		})

		Convey("2. test NewDeviceMap vgcu device success\n", func() {
			fakeDeviceCount, deviceType := 8, "vgcu"
			fakeDeviceList, err := tests.CreateFakeDevice(deviceType, fakeDeviceCount)
			defer func() {
				So(tests.ClearFakeDevice(fakeDeviceList), ShouldEqual, nil)
			}()
			So(err, ShouldEqual, nil)

			res, err := NewDeviceMap(gcu.NewRegisterDeviceLib(config), config)
			So(err, ShouldEqual, nil)
			So(res.isEmpty(), ShouldEqual, false)
			So(len(res), ShouldEqual, 1)
			devices, ok := res["enflame.com/vgcu"]
			So(ok, ShouldEqual, true)
			So(len(devices), ShouldEqual, 8)
		})
	})
}
