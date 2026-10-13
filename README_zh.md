# Enflame GCU 设备插件 (enflame-device-plugin)

> **语言**: [English](README.md) | 中文

## 目录

- [概述](#概述)
- [项目结构](#项目结构)
- [安装](#安装)
  - [前置条件](#前置条件)
  - [从源码构建](#从源码构建)
  - [构建容器镜像](#构建容器镜像)
- [快速开始](#快速开始)
  - [使用 Helm 部署](#使用-helm-部署)
  - [使用静态清单部署](#使用静态清单部署)
  - [在 Pod 中申请 GCU](#在-pod-中申请-gcu)
- [核心特性](#核心特性)
- [版本兼容性](#版本兼容性)
- [硬件支持](#硬件支持)
- [用户手册](#用户手册)
- [故障排查](#故障排查)
- [许可证](#许可证)

## 概述

`enflame-device-plugin` 是面向 Enflame GCU（General Compute Unit，通用计算单元）的
Kubernetes [设备插件](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/)，
其定位与 nvidia-device-plugin 类似。它会发现每个节点上的 GCU，并将其作为可调度资源
上报给 kubelet，使 Pod 能够像申请 CPU 或内存那样申请 GCU。

该插件上报两种资源名称：

- `enflame.com/gcu` —— 整卡 GCU。
- `enflame.com/vgcu` —— 虚拟 GCU（vGCU）切片。

Pod 在 `resources.limits` 下申请这些资源，例如 `enflame.com/gcu: 8`。该插件提供：

- **整卡分配** —— 将每张物理 GCU 作为一个可调度的 `enflame.com/gcu` 资源上报。
- **按卡 / vGCU 分配** —— 将整卡虚拟化，上报 `enflame.com/vgcu` 切片以实现更细粒度的共享。
- **CDI 支持** —— 通过 [容器设备接口（Container Device Interface）](https://github.com/cncf-tags/container-device-interface) 注入设备。
- **Kubernetes 原生** —— 以 DaemonSet 运行，可通过 Helm chart 或静态清单部署。

## 项目结构

```
enflame-device-plugin/
├── k8s-device-plugin/          # 源码；`make all` 构建两个二进制：
│                               #   enflame-device-plugin + config-manager
├── deployments/                # 部署脚本、Helm chart 和清单
│   ├── deploy.sh               # 部署插件
│   ├── delete.sh               # 卸载插件
│   ├── driver-check.sh         # 校验 Enflame GCU 驱动
│   ├── helm/                   # Helm chart（helm/enflame-device-plugin/）
│   ├── static/                 # 静态 DaemonSet 清单（by-card、cdi）
│   ├── example/                # 示例 Pod（pod-gcu.yaml、pod-vgcu.yaml）
│   └── dockerfiles/            # 容器镜像构建文件
├── build.sh                    # 构建 + 打包入口
└── test.sh                     # 测试入口
```

## 安装

### 前置条件

- 一个 Kubernetes 集群。
- 节点具备 Enflame GCU 设备并已安装 Enflame GCU 驱动。
- 从源码构建需要 [Go](https://go.dev/) 1.22+。
- 构建容器镜像需要 Docker（或兼容的构建工具）。

### 从源码构建

```bash
# 构建两个二进制，并在 dist/ 下生成发布包
./build.sh all

# 清理构建产物
./build.sh clean
```

`./build.sh all` 会执行 `cd k8s-device-plugin; make all`，然后将
`enflame-device-plugin` 和 `config-manager` 二进制连同 `deployments/` 目录一起
打包到 `dist/` 下。

### 构建容器镜像

容器镜像构建文件位于 `deployments/dockerfiles/`。请在该目录下构建镜像，并推送到
Helm values 或静态清单所使用的镜像仓库。

## 快速开始

### 使用 Helm 部署

```bash
helm install enflame-device-plugin deployments/helm/enflame-device-plugin/
```

请在 chart 的 `values.yaml` 中查看并覆盖相关配置；可用选项参见
`deployments/helm/enflame-device-plugin/README.md`。

### 使用静态清单部署

```bash
# 整卡 / 默认部署
./deployments/deploy.sh

# 按卡 / vGCU 部署
kubectl apply -f deployments/static/enflame-device-plugin-by-card.yaml

# CDI 部署
kubectl apply -f deployments/static/enflame-device-plugin-cdi.yaml

# 卸载插件
./deployments/delete.sh
```

该插件以 DaemonSet 运行，会在每个具备 Enflame GCU 设备的节点上调度一个 Pod。

### 在 Pod 中申请 GCU

在 `resources.limits` 下申请一张整卡：

```yaml
resources:
  limits:
    enflame.com/gcu: 8
```

项目提供了完整示例：`deployments/example/pod-gcu.yaml` 用于申请整卡 GCU，
`deployments/example/pod-vgcu.yaml` 用于申请 vGCU。

## 核心特性

| 特性 | 说明 |
| --- | --- |
| 整卡分配 | 将每张物理 GCU 作为 `enflame.com/gcu` 可调度资源上报 |
| vGCU / 按卡 | 将整卡虚拟化为 `enflame.com/vgcu` 切片，实现更细粒度的共享 |
| CDI | 通过容器设备接口（Container Device Interface）注入设备 |
| Helm chart | 通过 `deployments/helm/enflame-device-plugin/` 进行可配置部署 |
| 静态清单 | 提供开箱即用的 by-card 与 CDI DaemonSet 清单 |
| 多架构 | 支持为多种 CPU 架构构建 |

## 版本兼容性

`enflame-device-plugin` 依赖节点上安装的 Enflame GCU 驱动来发现设备，因此其能力
与所部署的驱动相关联。请始终使插件版本与节点上的 GCU 驱动相匹配，并将其部署在
版本支持设备插件 API 的 Kubernetes 集群上。

## 硬件支持

支持提供 Enflame GCU 驱动的 Enflame GCU 设备。同一插件可在各代 GCU 产品上覆盖
整卡、vGCU/按卡以及 CDI 分配方式。

## 用户手册

| 文档 | 说明 |
| --- | --- |
| [deployments/README.md](deployments/README.md) | 部署指南（脚本、静态清单、示例） |
| [deployments/helm/enflame-device-plugin/README.md](deployments/helm/enflame-device-plugin/README.md) | Helm chart 配置与 values 参考 |

## 故障排查

- **节点上未出现 `enflame.com/gcu` 资源** —— 确认已安装 Enflame GCU 驱动，并运行
  `./deployments/driver-check.sh` 进行校验；确认插件 DaemonSet 的 Pod 已在该节点上
  调度并处于 Running 状态。
- **插件 DaemonSet 未就绪** —— 检查 Pod 日志，确认节点确实存在 GCU 设备，并确保
  Helm values / 静态清单引用的容器镜像可以拉取。
- **申请 GCU 时 Pod 持续 Pending** —— 检查所申请的资源
  （`enflame.com/gcu` 或 `enflame.com/vgcu`）是否已在某节点上上报，且该节点有
  足够的空闲设备满足 `resources.limits` 请求。

## 许可证

`enflame-device-plugin` 采用 **Apache License 2.0** 许可证。版权所有 (c) Enflame
Technology，保留所有权利。详见 [LICENSE](LICENSE)。
