#!/usr/bin/env bash

# Copyright 2023 The Kubernetes Authors.
# Copyright 2023 NVIDIA CORPORATION.
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

# A reference to the current directory where this script is located
CURRENT_DIR="$(cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd)"

set -ex
set -o pipefail

source "${CURRENT_DIR}/common.sh"

kind create cluster \
	--retain \
	--name "${KIND_CLUSTER_NAME}" \
	--image "${KIND_IMAGE}" \
	--config "${KIND_CLUSTER_CONFIG_PATH}"

# Unmount the masked /proc/driver/enflame to allow
# dynamically generated MIG devices to be discovered
docker exec -it "${KIND_CLUSTER_NAME}-worker" umount -R /proc/driver/enflame

# Install the enflame-container-toolkit.
# TODO: Once we have a more standard way to enable this we can remove this.
docker exec -it "${KIND_CLUSTER_NAME}-worker" bash -c "apt-get update && apt-get install -y gpg"

docker exec -it "${KIND_CLUSTER_NAME}-worker" bash -c " \
	curl -fsSL https://enflame.github.io/libenflame-container/gpgkey | gpg --dearmor -o /usr/share/keyrings/enflame-container-toolkit-keyring.gpg \
  && curl -s -L https://enflame.github.io/libenflame-container/experimental/deb/enflame-container-toolkit.list | \
    sed 's#deb https://#deb [signed-by=/usr/share/keyrings/enflame-container-toolkit-keyring.gpg] https://#g' | \
    tee /etc/apt/sources.list.d/enflame-container-toolkit.list \
  && \
    apt-get update \
  && apt-get install -y enflame-container-toolkit"

# We configure the ENFLAME Container Runtime to only trigger on the enflame.cdi.k8s.io annotation and enable CDI in containerd.
docker exec -it "${KIND_CLUSTER_NAME}-worker" bash -c "\
	enflame-ctk config --set enflame-container-runtime.modes.cdi.annotation-prefixes=enflame.cdi.k8s.io/ \
	&& \
	enflame-ctk runtime configure --runtime=containerd --cdi.enabled \
	&& \
	systemctl restart containerd"

# Add an enflame RuntimeClass.
# TODO: This could be included in the yaml files instead.
kubectl apply -f - <<EOF
apiVersion: node.k8s.io/v1
handler: enflame
kind: RuntimeClass
metadata:
  name: enflame
EOF
