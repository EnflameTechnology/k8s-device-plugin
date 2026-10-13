# ENFLAME GCU feature discovery

> Migrated from https://gitlab.com/enflame/kubernetes/gcu-feature-discovery

## Table of Contents

- [Overview](#overview)
- [Beta Version](#beta-version)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
  * [Node Feature Discovery (NFD)](#node-feature-discovery-nfd)
  * [Preparing your GCU Nodes](#preparing-your-gcu-nodes)
  * [Deploy ENFLAME GCU Feature Discovery (GFD)](#deploy-enflame-gcu-feature-discovery-gfd)
    + [Daemonset](#daemonset)
    + [Job](#job)
  * [Verifying Everything Works](#verifying-everything-works)
- [The GFD Command line interface](#the-gfd-command-line-interface)
- [Generated Labels](#generated-labels)
  * [MIG 'single' strategy](#mig-single-strategy)
  * [MIG 'mixed' strategy](#mig-mixed-strategy)
- [Deployment via `helm`](#deployment-via-helm)
    + [Deploying via `helm install` with a direct URL to the `helm` package](#deploying-via-helm-install-with-a-direct-url-to-the-helm-package)
- [Building and running locally on your native machine](#building-and-running-locally-on-your-native-machine)

## Overview

ENFLAME GCU Feature Discovery for Kubernetes is a software component that allows
you to automatically generate labels for the set of GCUs available on a node.
It leverages the [Node Feature
Discovery](https://github.com/kubernetes-sigs/node-feature-discovery)
to perform this labeling.

## Beta Version

This tool should be considered beta until it reaches `v1.0.0`. As such, we may
break the API before reaching `v1.0.0`, but we will setup a deprecation policy
to ease the transition.

## Prerequisites

The list of prerequisites for running the ENFLAME GCU Feature Discovery is
described below:
* enflame-docker version > 2.0 (see how to [install](https://github.com/ENFLAME/enflame-docker)
and it's [prerequisites](https://github.com/enflame/enflame-docker/wiki/Installation-\(version-2.0\)#prerequisites))
* docker configured with enflame as the [default runtime](https://github.com/ENFLAME/enflame-docker/wiki/Advanced-topics#default-runtime).
* Kubernetes version >= 1.10
* ENFLAME device plugin for Kubernetes (see how to [setup](https://github.com/ENFLAME/k8s-device-plugin))
* NFD deployed on each node you want to label with the local source configured
  * When deploying GCU feature discovery with helm (as described below) we provide a way to automatically deploy NFD for you
  * To deploy NFD yourself, please see https://github.com/kubernetes-sigs/node-feature-discovery

## Quick Start

The following assumes you have at least one node in your cluster with GCUs and
the standard ENFLAME [drivers](https://www.enflame.com/Download/index.aspx) have
already been installed on it.

### Node Feature Discovery (NFD)

The first step is to make sure that [Node Feature Discovery](https://github.com/kubernetes-sigs/node-feature-discovery)
is running on every node you want to label. ENFLAME GCU Feature Discovery use
the `local` source so be sure to mount volumes. See
https://github.com/kubernetes-sigs/node-feature-discovery for more details.

You also need to configure the `Node Feature Discovery` to only expose vendor
IDs in the PCI source. To do so, please refer to the Node Feature Discovery
documentation.

The following command will deploy NFD with the minimum required set of
parameters to run `gcu-feature-discovery`.

```shell
kubectl apply -f https://raw.githubusercontent.com/ENFLAME/k8s-device-plugin/v0.15.0/deployments/static/nfd.yaml
```

**Note:** This is a simple static daemonset meant to demonstrate the basic
features required of `node-feature-discovery` in order to successfully run
`gcu-feature-discovery`. Please see the instructions below for [Deployment via
`helm`](#deployment-via-helm) when deploying in a production setting.

### Preparing your GCU Nodes

The following steps need to be executed on all your GCU nodes.
This README assumes that the ENFLAME drivers and the `enflame-container-toolkit` have been pre-installed.
It also assumes that you have configured the `enflame-container-runtime` as the default low-level runtime to use.

Please see: https://docs.enflame.com/datacenter/cloud-native/container-toolkit/install-guide.html

### Deploy ENFLAME GCU Feature Discovery (GFD)

The next step is to run ENFLAME GCU Feature Discovery on each node as a Daemonset
or as a Job.

#### Daemonset

```shell
kubectl apply -f https://raw.githubusercontent.com/ENFLAME/k8s-device-plugin/v0.15.0/deployments/static/gcu-feature-discovery-daemonset.yaml
```

**Note:** This is a simple static daemonset meant to demonstrate the basic
features required of `gcu-feature-discovery`. Please see the instructions below
for [Deployment via `helm`](#deployment-via-helm) when deploying in a
production setting.

#### Job

You must change the `NODE_NAME` value in the template to match the name of the
node you want to label:

```shell
$ export NODE_NAME=<your-node-name>
$ curl https://raw.githubusercontent.com/ENFLAME/k8s-device-plugin/v0.15.0/deployments/static/gcu-feature-discovery-job.yaml.template \
    | sed "s/NODE_NAME/${NODE_NAME}/" > gcu-feature-discovery-job.yaml
$ kubectl apply -f gcu-feature-discovery-job.yaml
```

**Note:** This method should only be used for testing and not deployed in a
productions setting.

### Verifying Everything Works

With both NFD and GFD deployed and running, you should now be able to see GCU
related labels appearing on any nodes that have GCUs installed on them.

```
$ kubectl get nodes -o yaml
apiVersion: v1
items:
- apiVersion: v1
  kind: Node
  metadata:
    ...

    labels:
      enflame.com/tops.driver.major: "455"
      enflame.com/tops.driver.minor: "06"
      enflame.com/tops.driver.rev: ""
      enflame.com/tops.runtime.major: "11"
      enflame.com/tops.runtime.minor: "1"
      enflame.com/gcu.compute.major: "8"
      enflame.com/gcu.compute.minor: "0"
      enflame.com/gfd.timestamp: "1594644571"
      enflame.com/gcu.count: "1"
      enflame.com/gcu.family: ampere
      enflame.com/gcu.machine: ENFLAME DGX-2H
      enflame.com/gcu.memory: "39538"
      enflame.com/gcu.product: A100-SXM4-40GB
      ...
...

```

## The GFD Command line interface

Available options:
```
gcu-feature-discovery:
Usage:
  gcu-feature-discovery [--fail-on-init-error=<bool>] [--mig-strategy=<strategy>] [--oneshot | --sleep-interval=<seconds>] [--no-timestamp] [--output-file=<file> | -o <file>]
  gcu-feature-discovery -h | --help
  gcu-feature-discovery --version

Options:
  -h --help                       Show this help message and exit
  --version                       Display version and exit
  --oneshot                       Label once and exit
  --no-timestamp                  Do not add timestamp to the labels
  --fail-on-init-error=<bool>     Fail if there is an error during initialization of any label sources [Default: true]
  --sleep-interval=<seconds>      Time to sleep between labeling [Default: 60s]
  --mig-strategy=<strategy>       Strategy to use for MIG-related labels [Default: none]
  -o <file> --output-file=<file>  Path to output file
                                  [Default: /etc/kubernetes/node-feature-discovery/features.d/gfd]

Arguments:
  <strategy>: none | single | mixed

```

You can also use environment variables:

| Env Variable           | Option               | Example |
| ---------------------- | -------------------- | ------- |
| GFD_FAIL_ON_INIT_ERROR | --fail-on-init-error | true    |
| GFD_MIG_STRATEGY       | --mig-strategy       | none    |
| GFD_ONESHOT            | --oneshot            | TRUE    |
| GFD_NO_TIMESTAMP       | --no-timestamp       | TRUE    |
| GFD_OUTPUT_FILE        | --output-file        | output  |
| GFD_SLEEP_INTERVAL     | --sleep-interval     | 10s     |

Environment variables override the command line options if they conflict.

## Generated Labels

This is the list of the labels generated by ENFLAME GCU Feature Discovery and
their meaning:

| Label Name                     | Value Type | Meaning                                      | Example        |
| -------------------------------| ---------- | -------------------------------------------- | -------------- |
| enflame.com/tops.driver.major   | Integer    | Major of the version of ENFLAME driver        | 418            |
| enflame.com/tops.driver.minor   | Integer    | Minor of the version of ENFLAME driver        | 30             |
| enflame.com/tops.driver.rev     | Integer    | Revision of the version of ENFLAME driver     | 40             |
| enflame.com/tops.runtime.major  | Integer    | Major of the version of TOPS                 | 10             |
| enflame.com/tops.runtime.minor  | Integer    | Minor of the version of TOPS                 | 1              |
| enflame.com/gfd.timestamp       | Integer    | Timestamp of the generated labels (optional) | 1555019244     |
| enflame.com/gcu.compute.major   | Integer    | Major of the compute capabilities            | 3              |
| enflame.com/gcu.compute.minor   | Integer    | Minor of the compute capabilities            | 3              |
| enflame.com/gcu.count           | Integer    | Number of GCUs                               | 2              |
| enflame.com/gcu.family          | String     | Architecture family of the GCU               | kepler         |
| enflame.com/gcu.machine         | String     | Machine type                                 | DGX-1          |
| enflame.com/gcu.memory          | Integer    | Memory of the GCU in Mb                      | 2048           |
| enflame.com/gcu.product         | String     | Model of the GCU                             | GeForce-GT-710 |

Depending on the MIG strategy used, the following set of labels may also be
available (or override the default values for some of the labels listed above):

### MIG 'single' strategy

With this strategy, the single `enflame.com/gcu` label is overloaded to provide
information about MIG devices on the node, rather than full GCUs. This assumes
all GCUs on the node have been divided into identical partitions of the same
size. The example below shows info for a system with 8 full GCUs, each of which
is partitioned into 7 equal sized MIG devices (56 total).

| Label Name                          | Value Type | Meaning                                  | Example                   |
| ----------------------------------- | ---------- | ---------------------------------------- | ------------------------- |
| enflame.com/mig.strategy             | String     | MIG strategy in use                      | single                    |
| enflame.com/gcu.product (overridden) | String     | Model of the GCU (with MIG info added)   | A100-SXM4-40GB-MIG-1g.5gb |
| enflame.com/gcu.count   (overridden) | Integer    | Number of MIG devices                    | 56                        |
| enflame.com/gcu.memory  (overridden) | Integer    | Memory of each MIG device in Mb          | 5120                      |
| enflame.com/gcu.multiprocessors      | Integer    | Number of Multiprocessors for MIG device | 14                        |
| enflame.com/gcu.slices.gi            | Integer    | Number of GCU Instance slices            | 1                         |
| enflame.com/gcu.slices.ci            | Integer    | Number of Compute Instance slices        | 1                         |
| enflame.com/gcu.engines.copy         | Integer    | Number of DMA engines for MIG device     | 1                         |
| enflame.com/gcu.engines.decoder      | Integer    | Number of decoders for MIG device        | 1                         |
| enflame.com/gcu.engines.encoder      | Integer    | Number of encoders for MIG device        | 1                         |
| enflame.com/gcu.engines.jpeg         | Integer    | Number of JPEG engines for MIG device    | 0                         |
| enflame.com/gcu.engines.ofa          | Integer    | Number of OfA engines for MIG device     | 0                         |

### MIG 'mixed' strategy

With this strategy, a separate set of labels for each MIG device type is
generated. The name of each MIG device type is defines as follows:
```
MIG_TYPE=mig-<slice_count>g.<memory_size>.gb
e.g.  MIG_TYPE=mig-3g.20gb
```

| Label Name                           | Value Type | Meaning                                  | Example        |
| ------------------------------------ | ---------- | ---------------------------------------- | -------------- |
| enflame.com/mig.strategy              | String     | MIG strategy in use                      | mixed          |
| enflame.com/MIG\_TYPE.count           | Integer    | Number of MIG devices of this type       | 2              |
| enflame.com/MIG\_TYPE.memory          | Integer    | Memory of MIG device type in Mb          | 10240          |
| enflame.com/MIG\_TYPE.multiprocessors | Integer    | Number of Multiprocessors for MIG device | 14             |
| enflame.com/MIG\_TYPE.slices.ci       | Integer    | Number of GCU Instance slices            | 1              |
| enflame.com/MIG\_TYPE.slices.gi       | Integer    | Number of Compute Instance slices        | 1              |
| enflame.com/MIG\_TYPE.engines.copy    | Integer    | Number of DMA engines for MIG device     | 1              |
| enflame.com/MIG\_TYPE.engines.decoder | Integer    | Number of decoders for MIG device        | 1              |
| enflame.com/MIG\_TYPE.engines.encoder | Integer    | Number of encoders for MIG device        | 1              |
| enflame.com/MIG\_TYPE.engines.jpeg    | Integer    | Number of JPEG engines for MIG device    | 0              |
| enflame.com/MIG\_TYPE.engines.ofa     | Integer    | Number of OfA engines for MIG device     | 0              |

## Deployment via `helm`

The preferred method to deploy `gcu-feature-discovery` is as a daemonset using `helm`.
Instructions for installing `helm` can be found
[here](https://helm.sh/docs/intro/install/).

As of `v0.15.0`, the device plugin's helm chart has integrated support to deploy
[`gcu-feature-discovery`](https://gitlab.com/enflame/kubernetes/gcu-feature-discovery/-/tree/main)

When gcu-feature-discovery in deploying standalone, begin by setting up the
plugin's `helm` repository and updating it at follows:

```shell
$ helm repo add efdp https://enflame.github.io/k8s-device-plugin
$ helm repo update
```

Then verify that the latest release (`v0.15.0`) of the plugin is available
(Note that this includes the GFD chart):

```shell
$ helm search repo efdp --devel
NAME                     	  CHART VERSION  APP VERSION	DESCRIPTION
efdp/enflame-device-plugin	  0.15.0	 0.15.0		A Helm chart for ...
```

Once this repo is updated, you can begin installing packages from it to deploy
the `gcu-feature-discovery` component in standalone mode.

The most basic installation command without any options is then:

```
$ helm upgrade -i efdp efdp/enflame-device-plugin \
  --version 0.15.0 \
  --namespace gcu-feature-discovery \
  --create-namespace \
  --set devicePlugin.enabled=false
```

Disabling auto-deployment of NFD and running with a MIG strategy of 'mixed' in
the default namespace.

```shell
$ helm upgrade -i efdp efdp/enflame-device-plugin \
    --version=0.15.0 \
    --set allowDefaultNamespace=true \
    --set nfd.enabled=false \
    --set migStrategy=mixed \
    --set devicePlugin.enabled=false
```

**Note:** You only need the to pass the `--devel` flag to `helm search repo`
and the `--version` flag to `helm upgrade -i` if this is a pre-release
version (e.g. `<version>-rc.1`). Full releases will be listed without this.

### Deploying via `helm install` with a direct URL to the `helm` package

If you prefer not to install from the `enflame-device-plugin` `helm` repo, you can
run `helm install` directly against the tarball of the plugin's `helm` package.
The example below installs the same chart as the method above, except that
it uses a direct URL to the `helm` chart instead of via the `helm` repo.

Using the default values for the flags:

```shell
$ helm upgrade -i efdp \
    --namespace gcu-feature-discovery \
    --set devicePlugin.enabled=false \
    --create-namespace \
    https://enflame.github.io/k8s-device-plugin/stable/enflame-device-plugin-0.15.0.tgz
```

## Building and running locally on your native machine

Download the source code:

```shell
git clone https://github.com/ENFLAME/k8s-device-plugin
```

Get dependies:

```shell
make vendor
```

Build it:

```
make build
```

Run it:
```
./gcu-feature-discovery --output=$(pwd)/gfd
```
