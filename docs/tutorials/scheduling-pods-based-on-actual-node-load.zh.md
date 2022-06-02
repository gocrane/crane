# Crane-scheduler

## 概述
Crane-scheduler 是一组基于[scheduler framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/)的调度插件， 包含：

- [Dynamic scheduler：负载感知调度器插件](./dynamic-scheduler-plugin.md)

## 开始

### 安装 Prometheus
确保你的 Kubernetes 集群已安装 Prometheus。如果没有，请参考[Install Prometheus](https://github.com/gocrane/fadvisor/blob/main/README.md#prerequests).

### 配置 Prometheus 规则

1. 配置 Prometheus 的规则以获取预期的聚合数据：

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
    name: example-record
spec:
    groups:
    - name: cpu_mem_usage_active
        interval: 30s
        rules:
        - record: cpu_usage_active
        expr: 100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[30s])) * 100)
        - record: mem_usage_active
        expr: 100*(1-node_memory_MemAvailable_bytes/node_memory_MemTotal_bytes)
    - name: cpu-usage-5m
        interval: 5m
        rules:
        - record: cpu_usage_max_avg_1h
        expr: max_over_time(cpu_usage_avg_5m[1h])
        - record: cpu_usage_max_avg_1d
        expr: max_over_time(cpu_usage_avg_5m[1d])
    - name: cpu-usage-1m
        interval: 1m
        rules:
        - record: cpu_usage_avg_5m
        expr: avg_over_time(cpu_usage_active[5m])
    - name: mem-usage-5m
        interval: 5m
        rules:
        - record: mem_usage_max_avg_1h
        expr: max_over_time(mem_usage_avg_5m[1h])
        - record: mem_usage_max_avg_1d
        expr: max_over_time(mem_usage_avg_5m[1d])
    - name: mem-usage-1m
        interval: 1m
        rules:
        - record: mem_usage_avg_5m
        expr: avg_over_time(mem_usage_active[5m])
```
!!! warning "️Troubleshooting"

        Prometheus 的采样间隔必须小于30秒，不然可能会导致规则无法正常生效。如：`cpu_usage_active`。

2\. 更新 Prometheus 服务发现的配置，确保`node_exporters/telegraf`正在使用节点名称作为实例名称：

```yaml hl_lines="9-11"
    - job_name: kubernetes-node-exporter
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        insecure_skip_verify: true
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      scheme: https
      kubernetes_sd_configs:
      ...
      # Host name
      - source_labels: [__meta_kubernetes_node_name]
        target_label: instance
      ...
```

!!! note "Note"

      如果节点名称是本机IP，则可以跳过此步骤。

### 安装 Crane-scheduler
有两种选择：

- 安装 Crane-scheduler 作为第二个调度器
- 用 Crane-scheduler 替换原生 Kube-scheduler

#### 安装 Crane-scheduler 作为第二个调度器
=== "Main"

       ```bash
       helm repo add crane https://gocrane.github.io/helm-charts
       helm install scheduler -n crane-system --create-namespace --set global.prometheusAddr="REPLACE_ME_WITH_PROMETHEUS_ADDR" crane/scheduler
       ```

=== "Mirror"

       ```bash
       helm repo add crane https://finops-helm.pkg.coding.net/gocrane/gocrane
       helm install scheduler -n crane-system --create-namespace --set global.prometheusAddr="REPLACE_ME_WITH_PROMETHEUS_ADDR" crane/scheduler
       ```
#### 用 Crane-scheduler 替换原生 Kube-scheduler

1. 备份`/etc/kubernetes/manifests/kube-scheduler.yaml`
```bash
cp /etc/kubernetes/manifests/kube-scheduler.yaml /etc/kubernetes/
```
2. 通过修改 kube-scheduler 的配置文件（`scheduler-config.yaml` ) 启用动态调度插件并配置插件参数：
```yaml title="scheduler-config.yaml"
apiVersion: kubescheduler.config.k8s.io/v1beta2
kind: KubeSchedulerConfiguration
...
profiles:
- schedulerName: default-scheduler
 plugins:
   filter:
     enabled:
     - name: Dynamic
   score:
     enabled:
     - name: Dynamic
       weight: 3
 pluginConfig:
 - name: Dynamic
    args:
     policyConfigPath: /etc/kubernetes/policy.yaml
...
```
3. 新建`/etc/kubernetes/policy.yaml`，用作动态插件的调度策略：
 ```yaml title="/etc/kubernetes/policy.yaml"
  apiVersion: scheduler.policy.crane.io/v1alpha1
  kind: DynamicSchedulerPolicy
  spec:
    syncPolicy:
      ##cpu usage
      - name: cpu_usage_avg_5m
        period: 3m
      - name: cpu_usage_max_avg_1h
        period: 15m
      - name: cpu_usage_max_avg_1d
        period: 3h
      ##memory usage
      - name: mem_usage_avg_5m
        period: 3m
      - name: mem_usage_max_avg_1h
        period: 15m
      - name: mem_usage_max_avg_1d
        period: 3h

    predicate:
      ##cpu usage
      - name: cpu_usage_avg_5m
        maxLimitPecent: 0.65
      - name: cpu_usage_max_avg_1h
        maxLimitPecent: 0.75
      ##memory usage
      - name: mem_usage_avg_5m
        maxLimitPecent: 0.65
      - name: mem_usage_max_avg_1h
        maxLimitPecent: 0.75

    priority:
      ##cpu usage
      - name: cpu_usage_avg_5m
        weight: 0.2
      - name: cpu_usage_max_avg_1h
        weight: 0.3
      - name: cpu_usage_max_avg_1d
        weight: 0.5
      ##memory usage
      - name: mem_usage_avg_5m
        weight: 0.2
      - name: mem_usage_max_avg_1h
        weight: 0.3
      - name: mem_usage_max_avg_1d
        weight: 0.5

    hotValue:
      - timeRange: 5m
        count: 5
      - timeRange: 1m
        count: 2
 ```
 4. 修改`kube-scheduler.yaml`并用 Crane-scheduler的镜像替换 kube-scheduler 镜像：
 ```yaml title="kube-scheduler.yaml"
 ...
  image: docker.io/gocrane/crane-scheduler:0.0.23
 ...
 ```
 5. 安装[crane-scheduler-controller](https://github.com/gocrane/crane-scheduler/tree/main/deploy/controller)：
=== "Main"

      ```bash
        kubectl apply -f https://raw.githubusercontent.com/gocrane/crane-scheduler/main/deploy/controller/rbac.yaml
        kubectl apply -f https://raw.githubusercontent.com/gocrane/crane-scheduler/main/deploy/controller/deployment.yaml
      ```

=== "Mirror"

      ```bash
      kubectl apply -f https://finops.coding.net/p/gocrane/d/crane-scheduler/git/raw/main/deploy/controller/rbac.yaml?download=false
      kubectl apply -f https://finops.coding.net/p/gocrane/d/crane-scheduler/git/raw/main/deploy/controller/deployment.yaml?download=false
      ```

### 使用 Crane-scheduler 调度 Pod
使用以下示例测试 Crane-scheduler ：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cpu-stress
spec:
  selector:
    matchLabels:
      app: cpu-stress
  replicas: 1
  template:
    metadata:
      labels:
        app: cpu-stress
    spec:
      schedulerName: crane-scheduler
      hostNetwork: true
      tolerations:
      - key: node.kubernetes.io/network-unavailable
        operator: Exists
        effect: NoSchedule
      containers:
      - name: stress
        image: docker.io/gocrane/stress:latest
        command: ["stress", "-c", "1"]
        resources:
          requests:
            memory: "1Gi"
            cpu: "1"
          limits:
            memory: "1Gi"
            cpu: "1"
```
!!! Note

    如果想将`crane-scheduler`用作默认调度器，请将`crane-scheduler`更改为`default-scheduler`。

如果测试 pod 调度成功，将会有以下事件：
```bash
Type    Reason     Age   From             Message
----    ------     ----  ----             -------
Normal  Scheduled  28s   crane-scheduler  Successfully assigned default/cpu-stress-7669499b57-zmrgb to vm-162-247-ubuntu
```
