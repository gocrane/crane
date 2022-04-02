# Getting Started

## Installation

**Prerequisites**

- Kubernetes 1.18+
- Helm 3.1.0

**Helm Installation**

Please refer to Helm's [documentation](https://helm.sh/docs/intro/install/) for installation.

**Installing prometheus and grafana with helm chart**

!!! note
    If you already deployed prometheus, grafana in your environment, then skip this step.

Crane use prometheus to be the default metric provider. 

Using following command to install prometheus components: prometheus-server, node-exporter, kube-state-metrics.

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus -n crane-system --set pushgateway.enabled=false --set alertmanager.enabled=false --set server.persistentVolume.enabled=false -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/prometheus/override_values.yaml --create-namespace  prometheus-community/prometheus
```

Fadvisor use grafana to present cost estimates. Using following command to install a grafana.

```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm install grafana -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/grafana/override_values.yaml -n crane-system --create-namespace grafana/grafana
```

**Deploying Crane and Fadvisor**

```bash
helm repo add crane https://gocrane.github.io/helm-charts
helm install crane -n crane-system --create-namespace crane/crane
helm install fadvisor -n crane-system --create-namespace crane/fadvisor
```

**Verify Installation**

Check deployments are all available by running:

```bash
kubectl get deploy -n crane-system
```

The output is similar to:
```bash
NAME                                             READY   STATUS    RESTARTS   AGE
crane-agent-8h7df                                1/1     Running   0          119m
crane-agent-8qf5n                                1/1     Running   0          119m
crane-agent-h9h5d                                1/1     Running   0          119m
craned-5c69c684d8-dxmhw                          2/2     Running   0          20m
grafana-7fddd867b4-kdxv2                         1/1     Running   0          41m
metric-adapter-94b6f75b-k8h7z                    1/1     Running   0          119m
prometheus-kube-state-metrics-6dbc9cd6c9-dfmkw   1/1     Running   0          45m
prometheus-node-exporter-bfv74                   1/1     Running   0          45m
prometheus-node-exporter-s6zps                   1/1     Running   0          45m
prometheus-node-exporter-x5rnm                   1/1     Running   0          45m
prometheus-server-5966b646fd-g9vxl               2/2     Running   0          45m
```

you can see [this](https://github.com/gocrane/helm-charts) to learn more.

**Customize Installation**

Deploy `Crane` by apply YAML declaration.

```bash
git clone https://github.com/gocrane/crane.git
CRANE_LATEST_VERSION=$(curl -s https://api.github.com/repos/gocrane/crane/releases/latest | grep -oP '"tag_name": "\K(.*)(?=")')
git checkout $CRANE_LATEST_VERSION
kubectl apply -f deploy/manifests 
kubectl apply -f deploy/craned 
kubectl apply -f deploy/metric-adapter
```

The following command will configure prometheus http address for crane if you want to customize it. Specify `CUSTOMIZE_PROMETHEUS` if you have existing prometheus server.

```bash
export CUSTOMIZE_PROMETHEUS=
if [ $CUSTOMIZE_PROMETHEUS ]; then sed -i '' "s/http:\/\/prometheus-server.crane-system.svc.cluster.local:8080/${CUSTOMIZE_PROMETHEUS}/" deploy/craned/deployment.yaml ; fi
```

## Get your Kubernetes Cost Report

Get the Grafana URL to visit by running these commands in the same shell:

```bash
export POD_NAME=$(kubectl get pods --namespace crane-system -l "app.kubernetes.io/name=grafana,app.kubernetes.io/instance=grafana" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace crane-system port-forward $POD_NAME 3000
```

visit [Cost Report](http://127.0.0.1:3000/dashboards) here with account(admin:admin).

## Analytics and Recommendation

Crane supports analytics and give recommend advise for your k8s cluster.

Please follow [this guide](tutorials/analytics-and-recommendation.md) to learn more.

## RoadMap
Please see [this document](roadmaps/roadmap-1h-2022.md) to learn more.

## Contributing

Contributors are welcomed to join Crane project. Please check [CONTRIBUTING](./CONTRIBUTING.md) about how to contribute to this project.

## Code of Conduct
Crane adopts [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).
