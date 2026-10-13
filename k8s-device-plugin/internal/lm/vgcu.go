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
	"enflame/k8s-device-plugin/internal/vgcu"
	"fmt"
)

// vgcuLabeler manages VGCUs labels for the node
type vgcuLabeler struct {
	lib vgcu.Interface
}

// NewVGCULabeler creates a new VGP label manager using the provided vgcu library
// and config.
func NewVGCULabeler(vgcu vgcu.Interface) Labeler {
	return vgcuLabeler{lib: vgcu}
}

// Labels generates the VGCU labels for the node
func (manager vgcuLabeler) Labels() (Labels, error) {
	devices, err := manager.lib.Devices()
	if err != nil {
		return nil, fmt.Errorf("unable to get vGCU devices: %v", err)
	}
	labels := make(Labels)
	if len(devices) > 0 {
		labels["enflame.com/vgcu.present"] = "true"
	}
	for _, device := range devices {
		info, err := device.GetInfo()
		if err != nil {
			return nil, fmt.Errorf("error getting vGCU device info: %v", err)
		}
		labels["enflame.com/vgcu.host-driver-version"] = info.HostDriverVersion
		labels["enflame.com/vgcu.host-driver-branch"] = info.HostDriverBranch
	}
	return labels, nil
}
