/**
# Copyright (c) 2021-2022, NVIDIA CORPORATION.  All rights reserved.
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

package lm

import (
	"testing"

	"github.com/stretchr/testify/require"

	spec "enflame/k8s-device-plugin/api/config/v1"
	"enflame/k8s-device-plugin/internal/resource"
	rt "enflame/k8s-device-plugin/internal/resource/testing"
)

func TestMigStrategyNoneLabels(t *testing.T) {
	testCases := []struct {
		description    string
		devices        []resource.Device
		timeSlicing    spec.ReplicatedResources
		expectedError  bool
		expectedLabels Labels
	}{
		{
			description: "no devices returns empty labels",
		},
		{
			description: "single non-mig device returns non-mig (none) labels",
			devices: []resource.Device{
				rt.NewFullGCU(),
			},
			expectedLabels: Labels{
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "1",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL",
			},
		},
		{
			description: "sharing is applied to single device",
			devices: []resource.Device{
				rt.NewFullGCU(),
			},
			timeSlicing: spec.ReplicatedResources{
				Resources: []spec.ReplicatedResource{
					{
						Name:     "enflame.com/gcu",
						Replicas: 2,
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "2",
				"enflame.com/gcu.sharing-strategy": "time-slicing",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL-SHARED",
			},
		},
		{
			description: "sharing is applied to multiple devices",
			devices: []resource.Device{
				rt.NewFullGCU(),
				rt.NewFullGCU(),
			},
			timeSlicing: spec.ReplicatedResources{
				Resources: []spec.ReplicatedResource{
					{
						Name:     "enflame.com/gcu",
						Replicas: 2,
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.count":            "2",
				"enflame.com/gcu.replicas":         "2",
				"enflame.com/gcu.sharing-strategy": "time-slicing",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL-SHARED",
			},
		},
		{
			description: "sharing is not applied to single MIG device; replicas is zero",
			devices: []resource.Device{
				rt.NewMigEnabledDevice(),
			},
			timeSlicing: spec.ReplicatedResources{
				Resources: []spec.ReplicatedResource{
					{
						Name:     "enflame.com/gcu",
						Replicas: 2,
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "0",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL",
			},
		},
		{
			description: "sharing is not applied to multiple MIG device; replicas is zero",
			devices: []resource.Device{
				rt.NewMigEnabledDevice(),
				rt.NewMigEnabledDevice(),
			},
			timeSlicing: spec.ReplicatedResources{
				Resources: []spec.ReplicatedResource{
					{
						Name:     "enflame.com/gcu",
						Replicas: 2,
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "2",
				"enflame.com/gcu.replicas":         "0",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL",
			},
		},
		{
			description: "sharing is applied to MIG device and non-MIG device",
			devices: []resource.Device{
				rt.NewMigEnabledDevice(),
				rt.NewFullGCU(),
			},
			timeSlicing: spec.ReplicatedResources{
				Resources: []spec.ReplicatedResource{
					{
						Name:     "enflame.com/gcu",
						Replicas: 2,
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.count":            "2",
				"enflame.com/gcu.replicas":         "2",
				"enflame.com/gcu.sharing-strategy": "time-slicing",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL-SHARED",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			efmlMock := rt.NewManagerMockWithDevices(tc.devices...)

			config := spec.Config{
				Flags: spec.Flags{
					CommandLineFlags: spec.CommandLineFlags{
						MigStrategy: ptr(MigStrategyNone),
					},
				},
				Sharing: spec.Sharing{
					TimeSlicing: tc.timeSlicing,
				},
			}

			none, _ := NewResourceLabeler(efmlMock, &config)

			labels, err := none.Labels()
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.EqualValues(t, tc.expectedLabels, labels)
		})
	}
}

func TestMigStrategySingleLabels(t *testing.T) {
	testCases := []struct {
		description    string
		devices        []resource.Device
		expectedError  bool
		expectedLabels Labels
		isInvalid      bool
	}{
		{
			description: "no devices returns empty labels",
		},
		{
			description: "single non-mig device returns non-mig (none) labels",
			devices: []resource.Device{
				rt.NewFullGCU(),
			},
			expectedLabels: Labels{
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "1",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL",
				"enflame.com/mig.strategy":         "single",
			},
		},
		{
			description: "multiple non-mig device returns non-mig (none) labels",
			devices: []resource.Device{
				rt.NewFullGCU(),
				rt.NewFullGCU(),
			},
			expectedLabels: Labels{
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.count":            "2",
				"enflame.com/gcu.replicas":         "1",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL",
				"enflame.com/mig.strategy":         "single",
			},
		},
		{
			description: "single mig-enabled device returns mig labels",
			devices: []resource.Device{
				rt.NewMigEnabledDevice(
					rt.NewMigDevice(1, 2, 100),
				),
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "1",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "100",
				"enflame.com/gcu.product":          "MOCKMODEL-MIG-1g.100gb",
				"enflame.com/mig.strategy":         "single",
				"enflame.com/gcu.multiprocessors":  "0",
				"enflame.com/gcu.slices.gi":        "1",
				"enflame.com/gcu.slices.ci":        "2",
				"enflame.com/gcu.engines.copy":     "0",
				"enflame.com/gcu.engines.decoder":  "0",
				"enflame.com/gcu.engines.encoder":  "0",
				"enflame.com/gcu.engines.jpeg":     "0",
				"enflame.com/gcu.engines.ofa":      "0",
			},
		},
		{
			description: "multiple mig-enabled devices returns mig labels",
			devices: []resource.Device{
				rt.NewMigEnabledDevice(
					rt.NewMigDevice(1, 2, 100, map[string]interface{}{
						"multiprocessors": 12,
						"engines.copy":    13,
						"engines.decoder": 14,
						"engines.encoder": 15,
						"engines.jpeg":    16,
						"engines.ofa":     17,
					}),
				),
				rt.NewMigEnabledDevice(
					rt.NewMigDevice(1, 2, 100, map[string]interface{}{
						"multiprocessors": 12,
						"engines.copy":    13,
						"engines.decoder": 14,
						"engines.encoder": 15,
						"engines.jpeg":    16,
						"engines.ofa":     17,
					}),
				),
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "2",
				"enflame.com/gcu.replicas":         "1",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "100",
				"enflame.com/gcu.product":          "MOCKMODEL-MIG-1g.100gb",
				"enflame.com/mig.strategy":         "single",
				"enflame.com/gcu.multiprocessors":  "12",
				"enflame.com/gcu.slices.gi":        "1",
				"enflame.com/gcu.slices.ci":        "2",
				"enflame.com/gcu.engines.copy":     "13",
				"enflame.com/gcu.engines.decoder":  "14",
				"enflame.com/gcu.engines.encoder":  "15",
				"enflame.com/gcu.engines.jpeg":     "16",
				"enflame.com/gcu.engines.ofa":      "17",
			},
		},
		{
			description: "empty mig devices returns MIG invalid label",
			devices: []resource.Device{
				rt.NewMigEnabledDevice(),
			},
			isInvalid: true,
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "0",
				"enflame.com/gcu.replicas":         "0",
				"enflame.com/gcu.sharing-strategy": "",
				"enflame.com/gcu.memory":           "0",
				"enflame.com/gcu.product":          "MOCKMODEL-MIG-INVALID",
				"enflame.com/mig.strategy":         "single",
			},
		},
		{
			description: "mixed mig config returns MIG invalid label",
			devices: []resource.Device{
				rt.NewMigEnabledDevice(
					rt.NewMigDevice(1, 2, 100),
					rt.NewMigDevice(3, 4, 100),
				),
			},
			isInvalid: true,
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "0",
				"enflame.com/gcu.replicas":         "0",
				"enflame.com/gcu.sharing-strategy": "",
				"enflame.com/gcu.memory":           "0",
				"enflame.com/gcu.product":          "MOCKMODEL-MIG-INVALID",
				"enflame.com/mig.strategy":         "single",
			},
		},
		{
			description: "mixed mig enabled and disabled returns invalid config",
			devices: []resource.Device{
				rt.NewMigEnabledDevice(
					rt.NewMigDevice(1, 2, 100),
				),
				rt.NewFullGCU(),
			},
			isInvalid: true,
			expectedLabels: Labels{
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.count":            "0",
				"enflame.com/gcu.replicas":         "0",
				"enflame.com/gcu.sharing-strategy": "",
				"enflame.com/gcu.memory":           "0",
				"enflame.com/gcu.product":          "MOCKMODEL-MIG-INVALID",
				"enflame.com/mig.strategy":         "single",
			},
		},
		{
			description: "enabled, disabled, and empty returns invalid config",
			devices: []resource.Device{
				rt.NewMigEnabledDevice(
					rt.NewMigDevice(1, 2, 100),
				),
				rt.NewFullGCU(),
				rt.NewMigEnabledDevice(),
			},
			isInvalid: true,
			expectedLabels: Labels{
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.count":            "0",
				"enflame.com/gcu.replicas":         "0",
				"enflame.com/gcu.sharing-strategy": "",
				"enflame.com/gcu.memory":           "0",
				"enflame.com/gcu.product":          "MOCKMODEL-MIG-INVALID",
				"enflame.com/mig.strategy":         "single",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			efmlMock := rt.NewManagerMockWithDevices(tc.devices...)

			config := spec.Config{
				Flags: spec.Flags{
					CommandLineFlags: spec.CommandLineFlags{
						MigStrategy: ptr(MigStrategySingle),
					},
				},
			}

			single, _ := NewResourceLabeler(efmlMock, &config)

			labels, err := single.Labels()
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.EqualValues(t, tc.expectedLabels, labels)
		})
	}
}

// prt returns a reference to whatever type is passed into it
func ptr[T any](x T) *T {
	return &x
}
