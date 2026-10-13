// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package gcu

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	v1 "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/tests"
)

func TestRegisterDeviceLib(t *testing.T) {
	Convey("test NewRegisterDeviceLib\n", t, func() {
		fakeDeviceCount, deviceType := 2, "gcu"
		fakeDeviceList, err := tests.CreateFakeDevice(deviceType, fakeDeviceCount)
		defer func() {
			So(tests.ClearFakeDevice(fakeDeviceList), ShouldEqual, nil)
		}()
		So(err, ShouldEqual, nil)

		fakeDeviceCount, deviceType = 4, "vgcu"
		fakeDeviceList, err = tests.CreateFakeDevice(deviceType, fakeDeviceCount)
		defer func() {
			So(tests.ClearFakeDevice(fakeDeviceList), ShouldEqual, nil)
		}()
		So(err, ShouldEqual, nil)

		config := &v1.Config{
			Resources: v1.Resources{
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
			},
		}
		deviceLib := NewRegisterDeviceLib(config)
		Convey("1. test NewRegisterDeviceLib device success\n", func() {
			So(len(deviceLib), ShouldEqual, 2)
			_, ok := deviceLib["gcu"]
			So(ok, ShouldEqual, true)
			_, ok = deviceLib["vgcu"]
			So(ok, ShouldEqual, true)
		})

		Convey("2. test GetDeviceIDPathMap\n", func() {
			for index := range deviceLib["gcu"].GetDeviceIDPathMap() {
				So(deviceLib["gcu"].GetDeviceIDPathMap()[index], ShouldEqual, fmt.Sprintf("/dev/gcu%d", index))
			}
			for index := range deviceLib["vgcu"].GetDeviceIDPathMap() {
				So(deviceLib["vgcu"].GetDeviceIDPathMap()[index], ShouldEqual, fmt.Sprintf("/dev/vgcu%d", index))
			}
		})

		Convey("3. test DeviceGetHandleByIndex\n", func() {
			Convey("3.1. test DeviceType\n", func() {
				for index := range len(deviceLib["gcu"].GetDeviceIDPathMap()) {
					So(deviceLib["gcu"].DeviceGetHandleByIndex(index).DeviceType(), ShouldEqual, "gcu")
				}
				for index := range len(deviceLib["vgcu"].GetDeviceIDPathMap()) {
					So(deviceLib["vgcu"].DeviceGetHandleByIndex(index).DeviceType(), ShouldEqual, "vgcu")
				}
			})

			Convey("3.2. test GetUUID\n", func() {
				for index := range len(deviceLib["gcu"].GetDeviceIDPathMap()) {
					So(deviceLib["gcu"].DeviceGetHandleByIndex(index).GetUUID(), ShouldEqual, fmt.Sprintf("GCU-%d", index))
				}
				for index := range len(deviceLib["vgcu"].GetDeviceIDPathMap()) {
					So(deviceLib["vgcu"].DeviceGetHandleByIndex(index).GetUUID(), ShouldEqual, fmt.Sprintf("VGCU-%d", index))
				}
			})

			Convey("3.3. test GetName\n", func() {
				for index := range len(deviceLib["gcu"].GetDeviceIDPathMap()) {
					So(deviceLib["gcu"].DeviceGetHandleByIndex(index).GetName(), ShouldEqual, fmt.Sprintf("gcu%d", index))
				}
				for index := range len(deviceLib["vgcu"].GetDeviceIDPathMap()) {
					So(deviceLib["vgcu"].DeviceGetHandleByIndex(index).GetName(), ShouldEqual, fmt.Sprintf("vgcu%d", index))
				}
			})

			Convey("3.4. test GetMinorNumber\n", func() {
				for index := range len(deviceLib["gcu"].GetDeviceIDPathMap()) {
					minor, err := deviceLib["gcu"].DeviceGetHandleByIndex(index).GetMinorNumber()
					So(err, ShouldEqual, nil)
					So(minor, ShouldEqual, 0)
				}
				for index := range len(deviceLib["vgcu"].GetDeviceIDPathMap()) {
					minor, err := deviceLib["vgcu"].DeviceGetHandleByIndex(index).GetMinorNumber()
					So(err, ShouldEqual, nil)
					So(minor, ShouldEqual, 0)
				}
			})

			Convey("3.5. test GetPaths\n", func() {
				for index := range len(deviceLib["gcu"].GetDeviceIDPathMap()) {
					paths, err := deviceLib["gcu"].DeviceGetHandleByIndex(index).GetPaths()
					So(err, ShouldEqual, nil)
					So(paths, ShouldResemble, []string{"/dev/gcu0"})
				}
				for index := range len(deviceLib["vgcu"].GetDeviceIDPathMap()) {
					paths, err := deviceLib["vgcu"].DeviceGetHandleByIndex(index).GetPaths()
					So(err, ShouldEqual, nil)
					So(paths, ShouldResemble, []string{"/dev/vgcu0"})
				}
			})
		})
	})
}
