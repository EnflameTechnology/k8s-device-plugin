#!/bin/bash -e

# Copyright (c) 2023, NVIDIA CORPORATION.  All rights reserved.
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

set -o pipefail

# Create temporary directory for GFD Helm chart
rm -rf deployments/helm/gcu-feature-discovery
mkdir -p deployments/helm/gcu-feature-discovery
cp -r deployments/helm/enflame-device-plugin/* deployments/helm/gcu-feature-discovery/
yq e -i '.devicePlugin.enabled = false | .gfd.enabled = true'  deployments/helm/gcu-feature-discovery/values.yaml
yq e -i '.name = "gcu-feature-discovery" | .description = "A Helm chart for gcu-feature-discovery on Kubernetes"' deployments/helm/gcu-feature-discovery/Chart.yaml

# Create release assets to be uploaded
helm package deployments/helm/gcu-feature-discovery/
helm package deployments/helm/enflame-device-plugin/
