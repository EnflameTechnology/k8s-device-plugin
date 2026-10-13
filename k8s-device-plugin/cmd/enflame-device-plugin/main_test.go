// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	. "github.com/agiledragon/gomonkey"
	"github.com/fsnotify/fsnotify"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	v1 "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/tests"
	"enflame/k8s-device-plugin/internal/utils"
)

func TestMain(t *testing.T) {
	Convey("test main Start\n", t, func() {
		fakeDeviceCount, deviceType := 8, "gcu"
		fakeDeviceList, err := tests.CreateFakeDevice(deviceType, fakeDeviceCount)
		defer func() {
			So(tests.ClearFakeDevice(fakeDeviceList), ShouldEqual, nil)
		}()
		So(err, ShouldEqual, nil)

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
				"cdiConfigPath": "/etc/cdi",
				"plugin": {
					"passDeviceSpecs": false,
					"pcieSwitchAffinity": false,
					"deviceListStrategy": [
						"envvar"
					],
					"deviceIDStrategy": "index",
					"cdiAnnotationPrefix": "cdi.k8s.io/",
					"enflameCTKPath": "/usr/bin/enflame-ctk",
					"containerDriverRoot": "/driver-root",
					"customRegisterResource": [
						"enflame.com/gcu"
					],
					"registerAllCdiConfig": true
				}
			}
		}
		`

		cdiConfig := `
		{
			"cdiVersion": "0.5.0",
			"containerEdits": {
			  "deviceNodes": [
				{
				  "path": "/dev/gcuctl"
				}
			  ],
			  "mounts": [
				{
				  "containerPath": "/lib/firmware/enflame_ucode_bin",
				  "hostPath": "/lib/firmware/enflame_ucode_bin",
				  "options": [
					"ro",
					"nosuid",
					"nodev",
					"bind"
				  ],
				  "type": "bind"
				}
			  ]
			},
			"devices": [
			  {
				"containerEdits": {
				  "deviceNodes": [
					{
					  "path": "/dev/gcu0"
					}
				  ]
				},
				"name": "0"
			  },
			  {
				"containerEdits": {
				  "deviceNodes": [
					{
					  "path": "/dev/gcu1"
					}
				  ]
				},
				"name": "1"
			  },
			  {
				"containerEdits": {
				  "deviceNodes": [
					{
					  "path": "/dev/gcu2"
					}
				  ]
				},
				"name": "2"
			  },
			  {
				"containerEdits": {
				  "deviceNodes": [
					{
					  "path": "/dev/gcu3"
					}
				  ]
				},
				"name": "3"
			  },
			  {
				"containerEdits": {
				  "deviceNodes": [
					{
					  "path": "/dev/gcu4"
					}
				  ]
				},
				"name": "4"
			  },
			  {
				"containerEdits": {
				  "deviceNodes": [
					{
					  "path": "/dev/gcu5"
					}
				  ]
				},
				"name": "5"
			  },
			  {
				"containerEdits": {
				  "deviceNodes": [
					{
					  "path": "/dev/gcu6"
					}
				  ]
				},
				"name": "6"
			  },
			  {
				"containerEdits": {
				  "deviceNodes": [
					{
					  "path": "/dev/gcu7"
					}
				  ]
				},
				"name": "7"
			  },
			  {
				"containerEdits": {
				  "deviceNodes": [
					{
					  "path": "/dev/gcu0"
					},
					{
					  "path": "/dev/gcu1"
					},
					{
					  "path": "/dev/gcu2"
					},
					{
					  "path": "/dev/gcu3"
					},
					{
					  "path": "/dev/gcu4"
					},
					{
					  "path": "/dev/gcu5"
					},
					{
					  "path": "/dev/gcu6"
					},
					{
					  "path": "/dev/gcu7"
					}
				  ]
				},
				"name": "all"
			  }
			],
			"kind": "enflame.com/gcu"
		  }
		`
		config := &v1.Config{}
		err = json.Unmarshal([]byte(configFile), config)
		So(err, ShouldEqual, nil)
		Convey("1. test main process success\n", func() {
			patcheAppRun := ApplyMethod(reflect.TypeOf(new(cli.App)), "Run", func(_ *cli.App, _ []string) error {
				return nil
			})
			defer patcheAppRun.Reset()
			main()
		})

		Convey("2. test startPlugins success\n", func() {
			patcheNewConfig := ApplyFunc(v1.NewConfig, func(_ *cli.Context, _ []cli.Flag) (*v1.Config, error) {
				return config, nil
			})
			defer patcheNewConfig.Reset()

			patcheReadFile := ApplyFunc(os.ReadFile, func(_ string) ([]byte, error) {
				return []byte("100"), nil
			})
			defer patcheReadFile.Reset()

			patcheNetListen := ApplyFunc(net.Listen, func(_ string, _ string) (net.Listener, error) {
				return nil, nil
			})
			defer patcheNetListen.Reset()

			patcheRegisterDevicePluginServer := ApplyFunc(pluginapi.RegisterDevicePluginServer, func(_ *grpc.Server,
				_ pluginapi.DevicePluginServer) {
			})
			defer patcheRegisterDevicePluginServer.Reset()

			patcheGrpcServe := ApplyMethod(reflect.TypeOf(new(grpc.Server)), "Serve", func(_ *grpc.Server, _ net.Listener) error {
				return nil
			})
			defer patcheGrpcServe.Reset()

			patcheGrpcDialContext := ApplyFunc(grpc.DialContext, func(_ context.Context, _ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
				return &grpc.ClientConn{}, nil
			})
			defer patcheGrpcDialContext.Reset()

			patcheGrpcClientClose := ApplyMethod(reflect.TypeOf(new(grpc.ClientConn)), "Close", func(_ *grpc.ClientConn) error {
				return nil
			})
			defer patcheGrpcClientClose.Reset()

			patcheClientRegister := ApplyFunc(pluginapi.NewRegistrationClient, func(_ *grpc.ClientConn) pluginapi.RegistrationClient {
				return &tests.RegistrationClient{}
			})
			defer patcheClientRegister.Reset()

			patcheExecCommand := ApplyFunc(utils.ExecCommand, func(_ string, _ ...string) error {
				return nil
			})
			defer patcheExecCommand.Reset()

			patcheReadYamlToJson := ApplyFunc(utils.ReadYamlToJson, func(_ string) ([]byte, error) {
				return []byte(cdiConfig), nil
			})
			defer patcheReadYamlToJson.Reset()

			watcher, err := fsnotify.NewWatcher()
			So(err, ShouldEqual, nil)
			defer func() {
				So(watcher.Close(), ShouldEqual, nil)
			}()
			So(os.MkdirAll(*config.Flags.CDIConfigPath, 0755), ShouldEqual, nil)
			defer func() {
				So(os.RemoveAll(*config.Flags.CDIConfigPath), ShouldEqual, nil)
			}()

			_, err = os.Create(*config.Flags.Plugin.EnflameCTKPath)
			So(err, ShouldEqual, nil)
			defer func() {
				So(os.Remove(*config.Flags.Plugin.EnflameCTKPath), ShouldEqual, nil)
			}()
			plugins, restart, err := startPlugins(config, watcher)
			// Waiting for the goroutine mock to take effect
			time.Sleep(1e5 * time.Microsecond)
			So(err, ShouldEqual, nil)
			So(len(plugins), ShouldEqual, 1)
			So(restart, ShouldEqual, false)
		})
	})
}
