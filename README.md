# Enflame GCU Device Plugin (enflame-device-plugin)

> **Language**: English | [中文](README_zh.md)

## Table of Contents

- [Overview](#overview)
- [Project Structure](#project-structure)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [Build from Source](#build-from-source)
  - [Build the Container Image](#build-the-container-image)
- [Quick Start](#quick-start)
  - [Deploy with Helm](#deploy-with-helm)
  - [Deploy with Static Manifests](#deploy-with-static-manifests)
  - [Request a GCU in a Pod](#request-a-gcu-in-a-pod)
- [Key Features](#key-features)
- [Version Compatibility](#version-compatibility)
- [Hardware Support](#hardware-support)
- [User Manual](#user-manual)
- [Trouble Shooting](#trouble-shooting)
- [License](#license)

## Overview

`enflame-device-plugin` is a Kubernetes [device plugin](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/)
for Enflame GCU (General Compute Unit), analogous to the nvidia-device-plugin.
It discovers the GCUs on each node and advertises them to the kubelet as
schedulable resources, so that pods can request GCUs the same way they request
CPU or memory.

The plugin advertises two resource names:

- `enflame.com/gcu` — a whole GCU card.
- `enflame.com/vgcu` — a virtual GCU (vGCU) slice.

A pod requests them under `resources.limits`, for example
`enflame.com/gcu: 8`. The plugin delivers:

- **Whole-card allocation** — expose each physical GCU as one schedulable `enflame.com/gcu` resource.
- **By-card / vGCU allocation** — virtualize a card and advertise `enflame.com/vgcu` slices for finer-grained sharing.
- **CDI support** — device injection through the [Container Device Interface](https://github.com/cncf-tags/container-device-interface).
- **Kubernetes-native** — runs as a DaemonSet, deployable with a Helm chart or static manifests.

## Project Structure

```
enflame-device-plugin/
├── k8s-device-plugin/          # Source; `make all` builds two binaries:
│                               #   enflame-device-plugin + config-manager
├── deployments/                # Deploy scripts, Helm chart and manifests
│   ├── deploy.sh               # Deploy the plugin
│   ├── delete.sh               # Remove the plugin
│   ├── driver-check.sh         # Validate the Enflame GCU driver
│   ├── helm/                   # Helm chart (helm/enflame-device-plugin/)
│   ├── static/                 # Static DaemonSet manifests (by-card, cdi)
│   ├── example/                # Example pods (pod-gcu.yaml, pod-vgcu.yaml)
│   └── dockerfiles/            # Container image build files
├── build.sh                    # Build + package entry point
└── test.sh                     # Test entry point
```

## Installation

### Prerequisites

- A Kubernetes cluster.
- Nodes with Enflame GCU devices and the Enflame GCU driver installed.
- [Go](https://go.dev/) 1.22+ for building from source.
- Docker (or a compatible builder) for building the container image.

### Build from Source

```bash
# Build both binaries and produce a release package under dist/
./build.sh all

# Clean build artifacts
./build.sh clean
```

`./build.sh all` runs `cd k8s-device-plugin; make all`, then packages the
`enflame-device-plugin` and `config-manager` binaries together with the
`deployments/` directory under `dist/`.

### Build the Container Image

Container image build files are provided under `deployments/dockerfiles/`.
Build the image from there and push it to the registry used by your Helm
values or static manifests.

## Quick Start

### Deploy with Helm

```bash
helm install enflame-device-plugin deployments/helm/enflame-device-plugin/
```

Review and override settings in the chart's `values.yaml`; see
`deployments/helm/enflame-device-plugin/README.md` for the available options.

### Deploy with Static Manifests

```bash
# Whole-card / default deployment
./deployments/deploy.sh

# By-card / vGCU deployment
kubectl apply -f deployments/static/enflame-device-plugin-by-card.yaml

# CDI deployment
kubectl apply -f deployments/static/enflame-device-plugin-cdi.yaml

# Remove the plugin
./deployments/delete.sh
```

The plugin runs as a DaemonSet, scheduling one pod onto every node that has
Enflame GCU devices.

### Request a GCU in a Pod

Request a whole card under `resources.limits`:

```yaml
resources:
  limits:
    enflame.com/gcu: 8
```

Complete examples are provided: `deployments/example/pod-gcu.yaml` for a whole
GCU, and `deployments/example/pod-vgcu.yaml` for a vGCU.

## Key Features

| Feature | Description |
| --- | --- |
| Whole-card allocation | Advertises each physical GCU as an `enflame.com/gcu` schedulable resource |
| vGCU / by-card | Virtualizes a card into `enflame.com/vgcu` slices for finer-grained sharing |
| CDI | Device injection through the Container Device Interface |
| Helm chart | Configurable deployment via `deployments/helm/enflame-device-plugin/` |
| Static manifests | Ready-to-apply by-card and CDI DaemonSet manifests |
| Multi-arch | Builds for multiple CPU architectures |

## Version Compatibility

`enflame-device-plugin` relies on the Enflame GCU driver installed on the node
to discover devices, so its capabilities track the deployed driver. Always
match the plugin release to the GCU driver on your nodes, and deploy it on a
Kubernetes cluster whose version supports the device plugin API.

## Hardware Support

Supports Enflame GCU devices that provide the Enflame GCU driver. The same
plugin covers whole-card, vGCU/by-card and CDI allocation across GCU product
generations.

## User Manual

| Document | Description |
| --- | --- |
| [deployments/README.md](deployments/README.md) | Deployment guide (scripts, static manifests, examples) |
| [deployments/helm/enflame-device-plugin/README.md](deployments/helm/enflame-device-plugin/README.md) | Helm chart configuration and values reference |

## Trouble Shooting

- **`enflame.com/gcu` resource does not show up on the node** — verify the
  Enflame GCU driver is installed and run `./deployments/driver-check.sh` to
  validate it; confirm the plugin DaemonSet pod is scheduled and Running on the
  node.
- **Plugin DaemonSet not Ready** — check the pod logs, confirm the node
  actually has GCU devices, and ensure the container image referenced by your
  Helm values / static manifest is reachable.
- **Pod stays Pending when requesting a GCU** — check that the requested
  resource (`enflame.com/gcu` or `enflame.com/vgcu`) is advertised on a node
  with enough free devices to satisfy the `resources.limits` request.

## License

`enflame-device-plugin` is licensed under the **Apache License 2.0**. Copyright
(c) Enflame Technology. All Rights Reserved. See [LICENSE](LICENSE).
