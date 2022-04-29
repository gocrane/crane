# Crane-scheduler

## Overview
Crane-scheduler is a collection of scheduler plugins based on [scheduler framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/), including:

- [Dynamic scheuler: a load-aware scheduler plugin](./dynamic-scheduler-plugin.md)

## Get Started

### 1. Install Prometheus
Make sure your kubernetes cluster has Prometheus installed. If not, please refer to [Install Prometheus](https://github.com/gocrane/fadvisor/blob/main/README.md#prerequests).

### 2. Configure Prometheus Rules
1) Configure the rules of Prometheus to get expected aggregated data:
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
>**⚠️Troubleshooting:** The sampling interval of Prometheus must be less than 30 seconds, otherwise the above rules(such as cpu_usage_active) may not take effect.
2) Update the configuration of Prometheus service discovery to ensure that node_exporters/telegraf are using node name as instance name:
```yaml
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
>**Note:** This step can be skipped if the node name itself is the host IP.

### 3. Install Crane-scheduler
There are two options:
1) Install Crane-scheduler as a second scheduler:
   ```bash
   helm repo add crane https://gocrane.github.io/helm-charts
   helm install scheduler -n crane-system --create-namespace --set global.prometheusAddr="REPLACE_ME_WITH_PROMETHEUS_ADDR" crane/scheduler
   ```
2) Replace native Kube-scheduler with Crane-scheduler:
   1) Backup `/etc/kubernetes/manifests/kube-scheduler.yaml`
   ```bash
   cp /etc/kubernetes/manifests/kube-scheduler.yaml /etc/kubernetes/
   ```
   2) Modify configfile of kube-scheduler(`scheduler-config.yaml`) to enable Dynamic scheduler plugin and configure plugin args:
   ```yaml
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
   3) Create `/etc/kubernetes/policy.yaml`, using as scheduler policy of Dynamic plugin:
   ```yaml
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
   4) Modify `kube-scheduler.yaml` and replace kube-scheduler image with Crane-scheduler：
   ```yaml
   ...
    image: docker.io/gocrane/crane-scheduler:0.0.23
   ...
   ```
   1) Install [crane-scheduler-controller](deploy/controller/deployment.yaml):
    ```bash
    kubectl apply ./deploy/controller/rbac.yaml && kubectl apply -f ./deploy/controller/deployment.yaml
    ```

### 4. Schedule Pods With Crane-scheduler
Test Crane-scheduler with following example:
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
>**Note:** Change `crane-scheduler` to `default-scheduler` if `crane-scheduler` is used as default.

There will be the following event if the test pod is successfully scheduled:
```bash
Type    Reason     Age   From             Message
----    ------     ----  ----             -------
Normal  Scheduled  28s   crane-scheduler  Successfully assigned default/cpu-stress-7669499b57-zmrgb to vm-162-247-ubuntu
```
