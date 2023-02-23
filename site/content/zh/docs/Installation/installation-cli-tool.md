---
title: "安装命令行工具"
description: "如何安装 kubectl-crane 命令行工具"
weight: 12
---

## 安装 kubectl-crane

你可以通过以下任意方式来安装 `kubectl-crane` 命令行工具

- 一键安装.
- 使用 krew 安装.
- 通过源码构建.

## 前提条件

- kubectl： `kubectl` 是 Kubernetes 命令行工具，可让您控制 Kubernetes 集群。
有关安装说明，请参阅 [安装 kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)。

### 一键安装

#### Linux

```shell
export release=v0.2.0
export arch=x86_64
curl -L -o kubectl-crane.tar.gz https://github.com/gocrane/kubectl-crane/releases/download/${release}/kubectl-crane_${release}_Linux_${arch}.tar.gz
tar -xvf kubectl-crane.tar.gz 
cp kubectl-crane_${release}_Linux_${arch}/kubectl-crane /usr/local/bin/
```

#### Mac

```shell
export release=v0.2.0
export arch=arm64
curl -L -o kubectl-crane.tar.gz https://github.com/gocrane/kubectl-crane/releases/download/${release}/kubectl-crane_${release}_Darwin_${arch}.tar.gz
tar -xvf kubectl-crane.tar.gz 
cp kubectl-crane_${release}_Darwin_${arch}/kubectl-crane /usr/local/bin/
```

### 使用 krew 安装

`Krew` 是 `kubectl` 命令行工具的插件管理器。

在你的机器上[安装和设置](https://krew.sigs.k8s.io/docs/user-guide/setup/install/) Krew。

然后安装 `kubectl-crane` 插件：

```shell
kubectl krew install crane
```

### 通过源码构建

```shell
git clone https://github.com/gocrane/kubectl-crane.git
cd kubectl-crane
export CGO_ENABLED=0
go mod vendor
go build -o kubectl-crane ./cmd/
```

然后将项目根目录下的 `kubectl-crane` 可执行文件移动到 `PATH` 路径下。
