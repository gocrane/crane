---
title: "安装文档"
description: "如何安装 Crane"
weight: 12
---

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

{{% alert color="warning" %}}
如果您已经在环境中部署了 Prometheus 和 Grafana，可以跳过该步骤。
{{% /alert %}}

{{% alert color="warning" %}}
如果你的网络无法访问GitHub资源(GitHub Release, GitHub Raw Content `raw.githubusercontent.com`)。
那么你可以尝试镜像仓库。但镜像仓库具有一定的**时延**。
{{% /alert %}}

Crane 使用 Prometheus 抓取集群工作负载对资源的使用情况。安装 Prometheus：

{{< tabpane right=true >}}
{{< tab header="Main" lang="en" >}}
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus -n crane-system \
    --set pushgateway.enabled=false \
    --set alertmanager.enabled=false \
    --set server.persistentVolume.enabled=false \
    -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/prometheus/override_values.yaml \
    --create-namespace  prometheus-community/prometheus
{{< /tab >}}
{{< tab header="Mirror" lang="en" >}}
helm repo add prometheus-community https://finops-helm.pkg.coding.net/gocrane/prometheus-community
helm install prometheus -n crane-system \
    --set pushgateway.enabled=false \
    --set alertmanager.enabled=false \
    --set server.persistentVolume.enabled=false \
    -f https://gitee.com/finops/helm-charts/raw/main/integration/prometheus/override_values.yaml \
    --create-namespace  prometheus-community/prometheus
{{< /tab >}}
{{% /tabpane %}}

Crane 的 Fadvisor 使用 Grafana 展示成本预估。安装 Grafana：

{{< tabpane right=true >}}
{{< tab header="Main" lang="en" >}}
helm repo add grafana https://grafana.github.io/helm-charts
helm install grafana \
    -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/grafana/override_values.yaml \
    -n crane-system \
    --create-namespace grafana/grafana
{{< /tab >}}
{{< tab header="Mirror" lang="en" >}}
helm repo add grafana https://finops-helm.pkg.coding.net/gocrane/grafana
helm install grafana \
    -f https://gitee.com/finops/helm-charts/raw/main/integration/grafana/override_values.yaml \
    -n crane-system \
    --create-namespace grafana/grafana
{{< /tab >}}
{{% /tabpane %}}

### 安装 Crane 和 Fadvisor

{{< tabpane right=true >}}
{{< tab header="Main" lang="en" >}}
helm repo add crane https://gocrane.github.io/helm-charts
helm install crane -n crane-system --create-namespace crane/crane
helm install fadvisor -n crane-system --create-namespace crane/fadvisor
{{< /tab >}}
{{< tab header="Mirror" lang="en" >}}
helm repo add crane https://finops-helm.pkg.coding.net/gocrane/gocrane
helm install crane -n crane-system --create-namespace crane/crane
helm install fadvisor -n crane-system --create-namespace crane/fadvisor
{{< /tab >}}
{{% /tabpane %}}

### 使用外部的 Prometheus（可选）

通常在生产环境，安装时需要配置外部的 Prometheus，你可以通过以下命令修改 Crane 的 Chart Release 配置或者直接修改 Craned Deployment 的容器 Args。

```bash
helm upgrade crane -n crane-system --set craned.containerArgs.prometheus-address=http://{prometheus-ip}:{port} --create-namespace crane/crane
```

同时，Crane Dashboard 的成本展示需要部署[kube-state-metrics](https://github.com/kubernetes/kube-state-metrics)（Prometheus Chart 中默认会安装），并且需要在你的 Prometheus 中配置额外的 extraScrapeConfigs，可以参考[这里](https://github.com/gocrane/helm-charts/blob/main/integration/prometheus/override_values.yaml#L56)。

最后，Fadvisor 需要配置 recording rules 来实现成本数据的聚合，可以参考[这里](https://github.com/gocrane/helm-charts/blob/main/integration/prometheus/override_values.yaml#L6)配置到你的 Prometheus 中。

### 使用外部的 Grafana（可选）

Crane Dashboard 支持通过 Iframe 内嵌 Grafana 报表展示成本分布。如果希望使用外部的 Grafana 内嵌到 Crane Dashboard，首先需要修改 configmap 中的 nginx 配置。

```bash
kubectl edit configmap -n crane-system nginx-conf
```

配置 `grafana.{{ .Release.Namespace }}.svc.cluster.local` 成外部的 Grafana 服务地址，配置 `http://$upstream_grafana:8082` 成外部的 Grafana 服务端口。

```yaml
 location /grafana {
    set $upstream_grafana grafana.{{ .Release.Namespace }}.svc.cluster.local;
    proxy_connect_timeout       180;
    proxy_send_timeout          180;
    proxy_read_timeout          180;
    proxy_pass http://$upstream_grafana:8082;
    proxy_redirect off;
    rewrite /grafana/(.*) /$1 break;
    proxy_http_version 1.1;
    proxy_set_header  Host $http_host;
    proxy_set_header  Upgrade $http_upgrade;
    proxy_set_header  Connection $connection_upgrade;
    proxy_set_header  X-Real-IP  $remote_addr;
    proxy_set_header  X-Forwarded-For $proxy_add_x_forwarded_for;
}

```

接下来需要参考[这里](https://github.com/gocrane/helm-charts/blob/main/integration/grafana/override_values.yaml)进行配置，原理是 Grafana 支持前端图表的内嵌，但是需要把对应的权限配置打开。

```bash
kubectl edit configmap -n monitor grafana
```

- 确定 Service 和 nginx 配置一致
- 配置 datasources 中的 prometheus 与你的环境一致
- 配置 dashboardProviders
- 配置 dashboards
- 配置 grafana.ini

最后，你需要确保 craned 和 grafana pods 已经重建并重新加载新的配置。

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

## 访问 Dashboard

用户可以通过 Dashboard 获取成本信息以及优化建议。

![](/images/dashboard.png)

### 端口映射

通过端口映射访问 Dashboard：

```bash
kubectl -n crane-system port-forward service/craned 9090:9090 
```

### NodePort

通过 NodePort 访问 Dashboard：

```bash
# Change service type
kubectl patch svc craned -n crane-system -p '{"spec": {"type": "NodePort"}}'
```

```bash
# Get Dashboard link base on your cluster configuration
PORT=$(kubectl get svc -n crane-system craned -o jsonpath='{.spec.ports[?(@.name == "dashboard-service")].nodePort}')
NODE_IP=$(kubectl get node -ojsonpath='{.items[].status.addresses[?(@.type == "InternalIP")].address}')
echo "Dashboard link: http://${NODE_IP}:${PORT}"
```

### LoadBalancer

通过 LoadBalancer 访问 Dashboard：

```bash
# Change service type
kubectl patch svc craned -n crane-system -p '{"spec": {"type": "LoadBalancer"}}'
```

```log
$ kubectl patch svc craned -n crane-system -p '{"spec": {"type": "LoadBalancer"}}'

service/craned patched

$ kubectl get svc -n crane-system craned
NAME     TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)                                                      AGE
craned   LoadBalancer   10.101.123.74   10.200.0.4    443:30908/TCP,8082:32426/TCP,9090:31331/TCP,8080:31072/TCP   57m

# Access dashboard via 10.200.0.4:9090
```

### Ingress

通过 Ingress 访问 Dashboard：

#### kubernetes/ingress-nginx

如果集群版本小于 1.19，可以创建以下 Ingress：

```yaml
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: ingress-crane-dashboard
  namespace: crane-system
spec:
  ingressClassName: nginx
  rules:
  - host: dashboard.gocrane.io # change to your domain
    http:
      paths:
      - path: /
        backend:
          serviceName: craned
          servicePort: 9090
```

如果集群版本大于等于 1.19，可以创建以下 Ingress：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-crane-dashboard
  namespace: crane-system
spec:
  rules:
  - host: dashboard.gocrane.io # change to your domain
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: craned
            port:
              number: 9090
  ingressClassName: nginx
```

例子:

```log
$ kubectl get svc -n ingress-nginx 
NAME                                 TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)                      AGE
ingress-nginx-controller             LoadBalancer   10.102.235.229   10.200.0.5    80:32568/TCP,443:30144/TCP   91m
ingress-nginx-controller-admission   ClusterIP      10.102.49.240    <none>        443/TCP                      91m

$ curl -H "Host: dashboard.gocrane.io" 10.200.0.5
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Crane Dashboard</title>
    ................................................................
```

#### Traefik

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: IngressRoute
metadata:
  name: dashboard-crane-ingress
  namespace: crane-system
spec:
  entryPoints:
    - web
  routes:
    - kind: Rule
      match: Host(`dashboard.gocrane.io`)
      services:
        - name: craned
          port: 9090
```

```log
$ kubectl get svc -n traefik-v2                     
NAME      TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)                      AGE
traefik   LoadBalancer   10.107.109.44   10.200.0.6    80:30102/TCP,443:30139/TCP   16m

$ curl -H "Host: dashboard.gocrane.io" 10.200.0.6 
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Crane Dashboard</title>
    ................................................................
```

## 自定义安装

通过 YAML 安装 `Crane` 。

{{< tabpane right=true >}}
{{< tab header="Main" lang="en" >}}
git clone https://github.com/gocrane/crane.git
CRANE_LATEST_VERSION=$(curl -s https://api.github.com/repos/gocrane/crane/releases/latest | grep -oP '"tag_name": "\K(.*)(?=")')
git checkout $CRANE_LATEST_VERSION
kubectl apply -f deploy/manifests
kubectl apply -f deploy/craned
kubectl apply -f deploy/metric-adapter
{{< /tab >}}
{{< tab header="Mirror" lang="en" >}}
git clone https://e.coding.net/finops/gocrane/crane.git
CRANE_LATEST_VERSION=$(curl -s https://api.github.com/repos/gocrane/crane/releases/latest | grep -oP '"tag_name": "\K(.*)(?=")')
git checkout $CRANE_LATEST_VERSION
kubectl apply -f deploy/manifests
kubectl apply -f deploy/craned
kubectl apply -f deploy/metric-adapter
{{< /tab >}}
{{% /tabpane %}}

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

![](/images/dashboard_nodeport.png)

注意：若您的集群节点只有内网 IP，则访问该 IP 的计算机需要在同一内网。若集群节点拥有外网 IP，则没有相关问题。

#### LoadBalance 方式

若您使用的是公有云厂商的服务，您可以将 Crane 的 Service 的类型换成公网 LB 类型，这样可以直接通过 LB IP + 9090 端口号的方式，打开控制台。

具体操作：修改 crane-system 命名空间下名为 craned 的 Service，将其访问方式该为公网 LB 的方式。
