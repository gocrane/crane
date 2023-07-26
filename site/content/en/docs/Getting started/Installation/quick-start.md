---
title: "Quick Start"
description: "Quick Start guide for Crane"
weight: 10
---

Welcome to Crane! In this document, we will work through how to install Crane and visit Crane Dashboard in your local
environments:

- Create a local running Kubernetes cluster by [Kind](https://kind.sigs.k8s.io/)
- Install Prometheus and Grafana by [Helm](https://helm.sh/)
- Install Crane by [Helm](https://helm.sh/)
- Visit Crane Dashboard via kubectl port-forward

Please referring to [Installation](/docs/getting-started/installation/installation/) to know more about how to install crane 。

## Prerequisites

- kubectl
- Kubernetes 1.18+
- Helm 3.1.0
- Kind 0.16+

{{% alert color="warning" %}}
If your Kubernetes version >= 1.26, please referring to [PR](https://github.com/gocrane/crane/pull/839)
{{% /alert %}}

## Installation

Following command will install Crane with dependencies applications(Prometheus/Grafana).

```bash
curl -sf https://raw.githubusercontent.com/gocrane/crane/main/hack/local-env-setup.sh | sh -
```

Make sure all pods are running:

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

## Visit Crane Dashboard

```bash
kubectl -n crane-system port-forward service/craned 9090:9090
```

Visit dashboard via [here](http://127.0.0.1:9090/)

![](/images/dashboard.png)
