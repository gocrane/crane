# Analytics and Recommendation

Analytics and Recommendation provide capacity that analyzes the workload in k8s cluster and provide recommendations about resource optimize.

Two Recommendations are currently supported:

- [**ResourceRecommend**](resource-recommendation.md): Recommend container requests & limit resources based on historic metrics.
- [**HPARecommend**](replicas-recommendation.md): Recommend which workloads are suitable for autoscaling and provide optimized configurations such as minReplicas, maxReplicas.

## Architecture

![analytics-arch](../images/analytics-arch.png)

## An analytical process

1. Users create `Analytics` object and config ResourceSelector to select resources to be analyzed. Multiple types of resource selection (based on Group,Kind, and Version) are supported. 
2. Analyze each selected resource in parallel and try to execute analysis and give recommendation. Each analysis process is divided into two stages: inspecting and advising:
     1. Inspecting: Filter resources that don't match the recommended conditions. For example, for hpa recommendation, the workload that has many not running pod is excluded
     2. Advising: Analysis and calculation based on algorithm model then provide the recommendation result.
3. If you paas the above two stages, it will create `Recommendation` object and display the result in `recommendation.Status`
4. You can find the failure reasons from `analytics.status.recommendations`
5. Wait for the next analytics based on the interval

## Core concept

### Analytics

Analysis defines a scanning analysis task. Two task types are supported: resource recommendation and hpa recommendation. Crane regularly runs analysis tasks and produces recommended results.

### Recommendation

The recommendation shows the results of an `Analytics`. The recommended result is a YAML configuration that allows users to take appropriate optimization actions, such as adjusting the resource configuration of the application.

### Configuration

Different analytics uses different computing models. Crane provides a default computing model and a corresponding configuration that users can modify to customize the recommended effect. You can modify the default configuration globally or modify the configuration of a single analytics task.
