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

package lm

import (
	"testing"

	"github.com/stretchr/testify/require"

	spec "enflame/k8s-device-plugin/api/config/v1"
	rt "enflame/k8s-device-plugin/internal/resource/testing"
)

func TestGCUResourceLabeler(t *testing.T) {
	device := rt.NewFullGCU()

	testCases := []struct {
		description    string
		count          int
		sharing        spec.Sharing
		expectedLabels Labels
	}{
		{
			description: "zero count returns empty",
		},
		{
			description: "no sharing",
			count:       1,
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "1",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
			},
		},
		{
			description: "time-slicing ignores non-matching resource",
			count:       1,
			sharing: spec.Sharing{
				TimeSlicing: spec.ReplicatedResources{
					Resources: []spec.ReplicatedResource{
						{
							Name:     "enflame.com/not-gcu",
							Replicas: 2,
						},
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "1",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
			},
		},
		{
			description: "time-slicing appends suffix and doubles count",
			count:       1,
			sharing: spec.Sharing{
				TimeSlicing: spec.ReplicatedResources{
					Resources: []spec.ReplicatedResource{
						{
							Name:     "enflame.com/gcu",
							Replicas: 2,
						},
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "2",
				"enflame.com/gcu.sharing-strategy": "time-slicing",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL-SHARED",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
			},
		},
		{
			description: "time-slicing renamed does not append suffix and doubles count",
			count:       1,
			sharing: spec.Sharing{
				TimeSlicing: spec.ReplicatedResources{
					Resources: []spec.ReplicatedResource{
						{
							Name:     "enflame.com/gcu",
							Rename:   "enflame.com/gcu.shared",
							Replicas: 2,
						},
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "2",
				"enflame.com/gcu.sharing-strategy": "time-slicing",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
			},
		},
		{
			description: "mps ignores non-matching resource",
			count:       1,
			sharing: spec.Sharing{
				MPS: &spec.ReplicatedResources{
					Resources: []spec.ReplicatedResource{
						{
							Name:     "enflame.com/not-gcu",
							Replicas: 2,
						},
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "1",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
			},
		},
		{
			description: "mps appends suffix and doubles count",
			count:       1,
			sharing: spec.Sharing{
				MPS: &spec.ReplicatedResources{
					Resources: []spec.ReplicatedResource{
						{
							Name:     "enflame.com/gcu",
							Replicas: 2,
						},
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "2",
				"enflame.com/gcu.sharing-strategy": "mps",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL-SHARED",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
			},
		},
		{
			description: "mps renamed does not append suffix and doubles count",
			count:       1,
			sharing: spec.Sharing{
				MPS: &spec.ReplicatedResources{
					Resources: []spec.ReplicatedResource{
						{
							Name:     "enflame.com/gcu",
							Rename:   "enflame.com/gcu.shared",
							Replicas: 2,
						},
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "2",
				"enflame.com/gcu.sharing-strategy": "mps",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL",
				"enflame.com/gcu.family":           "ampere",
				"enflame.com/gcu.compute.major":    "8",
				"enflame.com/gcu.compute.minor":    "0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			config := &spec.Config{
				Sharing: tc.sharing,
			}
			l, err := NewGCUResourceLabeler(config, device, tc.count)
			require.NoError(t, err)

			labels, err := l.Labels()
			require.NoError(t, err)

			require.EqualValues(t, tc.expectedLabels, labels)
		})
	}

}

func TestMigResourceLabeler(t *testing.T) {

	device := rt.NewMigDevice(1, 2, 300)
	rt.NewMigEnabledDevice(device)

	testCases := []struct {
		description    string
		resourceName   spec.ResourceName
		count          int
		timeSlicing    spec.ReplicatedResources
		expectedLabels Labels
	}{
		{
			description: "zero count returns empty",
		},
		{
			description:  "no sharing",
			resourceName: "enflame.com/gcu",
			count:        1,
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "1",
				"enflame.com/gcu.sharing-strategy": "none",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL-MIG-1g.300gb",
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
			description:  "shared appends suffix and doubles count",
			resourceName: "enflame.com/gcu",
			count:        1,
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
				"enflame.com/gcu.replicas":         "2",
				"enflame.com/gcu.sharing-strategy": "time-slicing",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL-MIG-1g.300gb-SHARED",
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
			description:  "renamed does not append suffix and doubles count",
			resourceName: "enflame.com/gcu",
			count:        1,
			timeSlicing: spec.ReplicatedResources{
				Resources: []spec.ReplicatedResource{
					{
						Name:     "enflame.com/gcu",
						Rename:   "enflame.com/gcu.shared",
						Replicas: 2,
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/gcu.count":            "1",
				"enflame.com/gcu.replicas":         "2",
				"enflame.com/gcu.sharing-strategy": "time-slicing",
				"enflame.com/gcu.memory":           "300",
				"enflame.com/gcu.product":          "MOCKMODEL-MIG-1g.300gb",
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
			description:  "mig mixed appends shared",
			resourceName: "enflame.com/mig-1g.1gb",
			count:        1,
			timeSlicing: spec.ReplicatedResources{
				Resources: []spec.ReplicatedResource{
					{
						Name:     "enflame.com/gcu",
						Rename:   "enflame.com/gcu.shared",
						Replicas: 2,
					},
					{
						Name:     "enflame.com/mig-1g.1gb",
						Replicas: 2,
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/mig-1g.1gb.count":            "1",
				"enflame.com/mig-1g.1gb.replicas":         "2",
				"enflame.com/mig-1g.1gb.sharing-strategy": "time-slicing",
				"enflame.com/mig-1g.1gb.memory":           "300",
				"enflame.com/mig-1g.1gb.product":          "MOCKMODEL-MIG-1g.300gb-SHARED",
				"enflame.com/mig-1g.1gb.multiprocessors":  "0",
				"enflame.com/mig-1g.1gb.slices.gi":        "1",
				"enflame.com/mig-1g.1gb.slices.ci":        "2",
				"enflame.com/mig-1g.1gb.engines.copy":     "0",
				"enflame.com/mig-1g.1gb.engines.decoder":  "0",
				"enflame.com/mig-1g.1gb.engines.encoder":  "0",
				"enflame.com/mig-1g.1gb.engines.jpeg":     "0",
				"enflame.com/mig-1g.1gb.engines.ofa":      "0",
			},
		},
		{
			description:  "mig mixed rename does not append",
			resourceName: "enflame.com/mig-1g.1gb",
			count:        1,
			timeSlicing: spec.ReplicatedResources{
				Resources: []spec.ReplicatedResource{
					{
						Name:     "enflame.com/mig-1g.1gb",
						Rename:   "enflame.com/mig-1g.1gb.shared",
						Replicas: 2,
					},
				},
			},
			expectedLabels: Labels{
				"enflame.com/mig-1g.1gb.count":            "1",
				"enflame.com/mig-1g.1gb.replicas":         "2",
				"enflame.com/mig-1g.1gb.sharing-strategy": "time-slicing",
				"enflame.com/mig-1g.1gb.memory":           "300",
				"enflame.com/mig-1g.1gb.product":          "MOCKMODEL-MIG-1g.300gb",
				"enflame.com/mig-1g.1gb.multiprocessors":  "0",
				"enflame.com/mig-1g.1gb.slices.gi":        "1",
				"enflame.com/mig-1g.1gb.slices.ci":        "2",
				"enflame.com/mig-1g.1gb.engines.copy":     "0",
				"enflame.com/mig-1g.1gb.engines.decoder":  "0",
				"enflame.com/mig-1g.1gb.engines.encoder":  "0",
				"enflame.com/mig-1g.1gb.engines.jpeg":     "0",
				"enflame.com/mig-1g.1gb.engines.ofa":      "0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			config := &spec.Config{
				Sharing: spec.Sharing{
					TimeSlicing: tc.timeSlicing,
				},
			}
			l, err := NewMIGResourceLabeler(tc.resourceName, config, device, tc.count)
			require.NoError(t, err)

			labels, err := l.Labels()
			require.NoError(t, err)

			require.EqualValues(t, tc.expectedLabels, labels)
		})
	}
}
