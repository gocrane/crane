---
title: "Installation of CLI Tool"
description: "How to install kubectl-crane"
weight: 11
---

## Install kubectl-crane

You can install `kubectl-crane` plugin in any of the following ways:

- One-click installation.
- Install using Krew.
- Build from source code.

## Prerequisites

- Kubectl: `kubectl` is the Kubernetes command line tool lets you control Kubernetes clusters.
For installation instructions see [installing kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl).

### One-click installation

#### For Linux

```shell
export release=v0.2.0
export arch=x86_64
curl -L -o kubectl-crane.tar.gz https://github.com/gocrane/kubectl-crane/releases/download/${release}/kubectl-crane_${release}_Linux_${arch}.tar.gz
tar -xvf kubectl-crane.tar.gz 
cp kubectl-crane_${release}_Linux_${arch}/kubectl-crane /usr/local/bin/
```

#### For Mac

```shell
export release=v0.2.0
export arch=arm64
curl -L -o kubectl-crane.tar.gz https://github.com/gocrane/kubectl-crane/releases/download/${release}/kubectl-crane_${release}_Darwin_${arch}.tar.gz
tar -xvf kubectl-crane.tar.gz 
cp kubectl-crane_${release}_Darwin_${arch}/kubectl-crane /usr/local/bin/
```

### Install using Krew

`Krew` is the plugin manager for `kubectl` command-line tool.

[Install and setup](https://krew.sigs.k8s.io/docs/user-guide/setup/install/) Krew on your machine.

Then install `kubectl-crane` plug-in:

```shell
kubectl krew install crane
```

### Build from source code

```shell
git clone https://github.com/gocrane/kubectl-crane.git
cd kubectl-crane
export CGO_ENABLED=0
go mod vendor
go build -o kubectl-crane ./cmd/
```

Next, move the `kubectl-crane` executable file in the project root directory to the `PATH` path.
