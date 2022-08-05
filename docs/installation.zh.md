# 产品部署指南

为了让您更快的部署 Crane ，本文档提供清晰的：

* 部署环境要求
* 具体安装步骤

Crane 安装时间在10分钟左右，具体时间也依赖集群规模以及硬件能力。目前安装已经非常成熟，如果您安装中遇到任何问题，可以采取如下几种方式：

* 请首先检查后文的 F&Q
* 可以提出一个 [Issue](https://github.com/gocrane/crane/issues/new?assignees=&labels=kind%2Fbug&template=bug_report.md&title=)，我们会认真对待每一个 [Issue](https://github.com/gocrane/crane/issues)

## 部署环境要求

- Kubernetes 1.18+
- Helm 3.1.0

## 安装流程

### 安装 Helm

建议参考 Helm 官网[安装文档](https://helm.sh/docs/intro/install/)。

### 安装 Prometheus 和 Grafana

使用 Helm 安装 Prometheus 和 Grafana。

!!! Note "注意" 
    如果您已经在环境中部署了 Prometheus 和 Grafana，可以跳过该步骤。

!!! Warning "网络问题"
    如果你的网络无法访问GitHub资源(GitHub Release, GitHub Raw Content `raw.githubusercontent.com`)。
    
    那么你可以尝试镜像仓库。但镜像仓库具有一定的**时延**。[镜像仓库](mirror.zh.md)


Crane 使用 Prometheus 抓取集群工作负载对资源的使用情况。安装 Prometheus：

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
                            -f https://gitee.com/finops/helm-charts/raw/main/integration/prometheus/override_values.yaml \
                            --create-namespace  prometheus-community/prometheus
    ```


Crane 的 Fadvisor 使用 Grafana 展示成本预估。安装 Grafana：

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
                 -f https://gitee.com/finops/helm-charts/raw/main/integration/grafana/override_values.yaml \
                 -n crane-system \
                 --create-namespace grafana/grafana
    ```

### 安装 Crane 和 Fadvisor

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

### 安装 Crane-scheduler（可选）
```console
helm install scheduler -n crane-system --create-namespace crane/scheduler
```

## 验证安装是否成功

使用如下命令检查安装的 Deployment 是否正常：

```console
kubectl get deploy -n crane-system
```

结果类似如下：

```shell
NAME                            READY   UP-TO-DATE   AVAILABLE   AGE
craned                          1/1     1            1           31m
fadvisor                        1/1     1            1           41m
grafana                         1/1     1            1           42m
metric-adapter                  1/1     1            1           31m
prometheus-kube-state-metrics   1/1     1            1           43m
prometheus-server               1/1     1            1           43m
```

可以查看本篇[文档](https://github.com/gocrane/helm-charts/blob/main/charts/crane/README.md)获取更多有关 Crane Helm Chart 的信息。

## 成本展示

### 打开 Crane 控制台

注意：Crane 的控制台地址就是 Crane 的 URL 地址，可以将其添加到统一的控制台查看多个部署 Crane 的集群的信息。

利用 [Port forwarding](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/) 命令，可以在本地计算机的浏览器打开 Crane 控制台：

```
kubectl port-forward -n crane-system svc/craned 9090
```

执行上述命令后，不要关闭命令行工具，在本地计算机的浏览器地址里输入 `localhost:9090`即可打开 Crane 的控制台：

![](images/crane-dashboard.png)

### 添加安装了 Crane 的集群

您可以点击上图中的“添加集群”的蓝色按钮，将 Crane 控制台的地址 `http://localhost:9090` 作为 Crane 的 URL，作为第一个集群添加到 Crane 控制台。

![](images/add_cluster.png)

若您想添加其它集群，实现多集群的资源使用和成本分析。可以在别的集群中也安装完 Crane 之后，将 Crane 的 URL 添加进来。

## 自定义安装

通过 YAML 安装 `Crane` 。

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

如果您想自定义 Crane 里配置 Prometheus 的 HTTP 地址，请参考以下的命令。如果您在集群里已存在一个 Prometheus，请将 Server 地址填于`CUSTOMIZE_PROMETHEUS` 。

```console
export CUSTOMIZE_PROMETHEUS=
if [ $CUSTOMIZE_PROMETHEUS ]; then sed -i '' "s/http:\/\/prometheus-server.crane-system.svc.cluster.local:8080/${CUSTOMIZE_PROMETHEUS}/" deploy/craned/deployment.yaml ; fi
```

## 安装常见问题

### 安装 Crane 报错

当您执行 `helm install crane -n crane-system --create-namespace crane/crane` 命令时，可能会遇到如下错误：

```shell
Error: rendered manifests contain a resource that already exists. Unable to continue with install: APIService "v1beta1.custom.metrics.k8s.io" in namespace "" exists and cannot be imported into the current release: invalid ownership metadata; label validation error: missing key "app.kubernetes.io/managed-by": must be set to "Helm"; annotation validation error: missing key "meta.helm.sh/release-name": must be set to "crane"; annotation validation error: missing key "meta.helm.sh/release-namespace": must be set to "crane-system"
```

原因：集群安装过 custom metric 的 APIService，所以报错。可以把之前的删除再重新执行安装 Crane 的命令，删除方式：`kubectl delete apiservice v1beta1.custom.metrics.k8s.io`。

### 获取 Crane URL 的其它方式

#### NodePort 方式

您可以将 Crane 的 Service 的类型换成 NodePort 类型，这样可以直接通过集群任意节点 IP + 该服务里dashboard- service 端口号的方式，打开控制台。

具体操作：修改 crane-system 命名空间下名为 craned 的 Service，将其访问方式该为 NodePort 的方式，然后获取某一集群的节点 IP，以及相应的端口号，端口号如下所示：

![](images/dashboard_nodeport.png)

注意：若您的集群节点只有内网 IP，则访问该 IP 的计算机需要在同一内网。若集群节点拥有外网 IP，则没有相关问题。

#### LoadBalance 方式

若您使用的是公有云厂商的服务，您可以将 Crane 的 Service 的类型换成公网 LB 类型，这样可以直接通过 LB IP + 9090 端口号的方式，打开控制台。

具体操作：修改 crane-system 命名空间下名为 craned 的 Service，将其访问方式该为公网 LB 的方式。
