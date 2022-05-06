# Installation

## Prerequisites

- Kubernetes 1.18+
- Helm 3.1.0

## Steps

### Helm Installation

Please refer to Helm's [documentation](https://helm.sh/docs/intro/install/) for installation.

### Installing prometheus and grafana with helm chart

!!! note
    If you already deployed prometheus, grafana in your environment, then skip this step.

!!! Warning "Network Problems"
    If your network is hard to connect GitHub resources, you can try the mirror repo. Like GitHub Release, GitHub Raw Content `raw.githubusercontent.com`.

    But mirror repo has a certain **latency**.[Mirror Repo](mirror.md)

Crane use prometheus to be the default metric provider. 

Using following command to install prometheus components: prometheus-server, node-exporter, kube-state-metrics.

=== "Main"

    ```bash
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm install prometheus -n crane-system \
                            --set pushgateway.enabled=false \
                            --set alertmanager.enabled=false \
                            --set server.persistentVolume.enabled=false \
                            -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/prometheus/override_values.yaml \
                            --create-namespace  prometheus-community/prometheus
    ```

=== "Mirror"

    ```bash
    helm repo add prometheus-community https://finops-helm.pkg.coding.net/gocrane/prometheus-community
    helm install prometheus -n crane-system \
                            --set pushgateway.enabled=false \
                            --set alertmanager.enabled=false \
                            --set server.persistentVolume.enabled=false \
                            -f https://finops.coding.net/p/gocrane/d/helm-charts/git/raw/main/integration/prometheus/override_values.yaml?download=false \
                            --create-namespace  prometheus-community/prometheus
    ```
Fadvisor use grafana to present cost estimates. Using following command to install a grafana.


=== "Main"

    ```bash
    helm repo add grafana https://grafana.github.io/helm-charts
    helm install grafana \
                 -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/grafana/override_values.yaml \
                 -n crane-system \
                 --create-namespace grafana/grafana
    ```

=== "Mirror"

    ```bash
    helm repo add grafana https://finops-helm.pkg.coding.net/gocrane/grafana
    helm install grafana \
                 -f https://finops.coding.net/p/gocrane/d/helm-charts/git/raw/main/integration/grafana/override_values.yaml?download=false \
                 -n crane-system \
                 --create-namespace grafana/grafana
    ```

### Deploying Crane and Fadvisor


=== "Main"

    ```bash
    helm repo add crane https://gocrane.github.io/helm-charts
    helm install crane -n crane-system --create-namespace crane/crane
    helm install fadvisor -n crane-system --create-namespace crane/fadvisor
    ```

=== "Mirror"

    ```bash
    helm repo add crane https://finops-helm.pkg.coding.net/gocrane/gocrane
    helm install crane -n crane-system --create-namespace crane/crane
    helm install fadvisor -n crane-system --create-namespace crane/fadvisor
    ```

### Deploying Crane-scheduler(optional)
```bash
helm install scheduler -n crane-system --create-namespace crane/scheduler
```

### Verify Installation

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

## Customize Installation

Deploy `Crane` by apply YAML declaration.

=== "Main"

    ```bash
    git clone https://github.com/gocrane/crane.git
    CRANE_LATEST_VERSION=$(curl -s https://api.github.com/repos/gocrane/crane/releases/latest | grep -oP '"tag_name": "\K(.*)(?=")')
    git checkout $CRANE_LATEST_VERSION
    kubectl apply -f deploy/manifests 
    kubectl apply -f deploy/craned 
    kubectl apply -f deploy/metric-adapter
    ```

=== "Mirror"

    ```bash
    git clone https://e.coding.net/finops/gocrane/crane.git
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
