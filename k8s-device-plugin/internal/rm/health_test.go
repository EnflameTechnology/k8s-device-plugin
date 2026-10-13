/**
# Copyright (c) 2021, NVIDIA CORPORATION.  All rights reserved.
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
	"os"
	"testing"
	"time"

	// . "github.com/agiledragon/gomonkey"
	. "github.com/smartystreets/goconvey/convey"

	"enflame/k8s-device-plugin/internal/tests"
)

func TestWatchHealth(t *testing.T) {
	Convey("test watchHealth\n", t, func() {
		fakeDeviceCount, deviceType := 8, "gcu"
		fakeDeviceList, err := tests.CreateFakeDevice(deviceType, fakeDeviceCount)
		defer func() {
			So(tests.ClearFakeDevice(fakeDeviceList), ShouldEqual, nil)
		}()
		So(err, ShouldEqual, nil)

		Convey("1. test disable health check\n", func() {
			err := os.Setenv(envDisableHealthChecks, "all")
			defer func() {
				err := os.Unsetenv(envDisableHealthChecks)
				So(err, ShouldEqual, nil)
			}()
			So(err, ShouldEqual, nil)
			watchHealth(nil, nil, nil, nil)
		})

		Convey("2. test health check stop\n", func() {
			stop := make(chan interface{})
			go func() { stop <- true }()
			// make sure goroutine execute first via sleep
			time.Sleep(1e5 * time.Microsecond)
			watchHealth(stop, nil, nil, nil)
		})
	})
}
