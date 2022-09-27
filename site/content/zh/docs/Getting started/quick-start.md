---
title: "快速开始"
description: "如何快速上手 Crane"
weight: 11
---

欢迎来到 Crane！在本文档中我们将介绍如何在本地安装 Crane 以及访问 Crane Dashboard：

- 使用 [Kind](https://kind.sigs.k8s.io/) 安装一个本地运行的 Kubernetes 集群
- 使用 [Helm](https://helm.sh/) 安装 Prometheus 和 Grafana
- 使用 [Helm](https://helm.sh/) 安装 Crane
- 通过 kubectl 的 port-forward 访问 Crane Dashboard

更多关于安装的介绍请参考 [安装文档](/zh-cn/docs/getting-started/installation) 。

## 部署环境要求

- kubectl
- Kubernetes 1.18+
- Helm 3.1.0
- Kind 0.16+

## 安装

以下命令将安装 Crane 以及其依赖 (Prometheus/Grafana).

```bash
curl -sf https://raw.githubusercontent.com/gocrane/crane/main/hack/local-env-setup.sh | sh -
```

确保所有 Pod 都正常运行:

```bash
$ export KUBECONFIG=${HOME}/.kube/config_crane
$ kubectl get deploy -n crane-system
NAME                                             READY   STATUS    RESTARTS       AGE
crane-agent-5r9l2                                1/1     Running   0              4m40s
craned-6dcc5c569f-vnfsf                          2/2     Running   0              4m41s
fadvisor-5b685f4cd6-xpxzq                        1/1     Running   0              4m37s
grafana-64656f6d54-6l24j                         1/1     Running   0              4m46s
metric-adapter-967c6d57f-swhfv                   1/1     Running   0              4m41s
prometheus-kube-state-metrics-7f9d78cffc-p8l7c   1/1     Running   0              4m46s
prometheus-node-exporter-4wk8b                   1/1     Running   0              4m40s
prometheus-server-fb944f4b7-4qqlv                2/2     Running   0              4m46s
```

## 访问 Crane Dashboard

```bash
kubectl -n crane-system port-forward service/craned 9090:9090
```

点击 [这里](http://127.0.0.1:9090/) 访问 Crane Dashboard

![](/images/dashboard.png)
