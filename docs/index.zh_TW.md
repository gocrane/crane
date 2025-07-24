# 介紹

Crane的目標是提供一站式專案，幫助Kubernetes使用者通過豐富的功能節省雲端資源的使用量：

- 基於監控資料的**時間序列預測**
- **資源使用率與成本的可視化**
- **使用量及成本優化** 包含：
    - R2 資源的重新分配(Resource Re-allocation)
    - R3 請求和副本的建議(Request & Replicas Recommendation)
    - 高效率的Pod自動彈性化
    - 成本最佳化
- 基於Pod優先級的**增強QoS**
- **負載感知調度**


![Crane Overview](images/crane-overview.png)

## 特色
### 時間序列預測


使用時間序列預測定義度量規範來預測 Kubernetes 資源，如Pod或Node。預測模組是其他Crane元件的核心元件，例如：[EHPA](#effective-horizontalpodautoscaler) 和 [Analytics](#analytics)

請參閱 [本文件](tutorials/using-time-series-prediction.md) 了解更多信息。

### 高效率的Pod自動縮放器

高效率的Pod自動縮放器幫助您輕鬆管理應用程式的擴展。 它與原生[HorizontalPodAutoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) 兼容，但擴展了更多功能，例如預測驅動的自動縮放。

請參閱 [本文件](tutorials/using-effective-hpa-to-scaling-with-effectiveness.md) 了解更多信息。

### 分析

分析模組分析工作負載並提出有關資源優化的建議。

目前支持兩個建議：
- [**ResourceRecommend**](tutorials/resource-recommendation.md): 副本推薦分析實際應用程式的使用情況，並為副本和 HPA 配置提供建議。
- [**HPARecommend**](tutorials/replicas-recommendation.md): 資源推薦可以讓您獲取叢集中資源的推薦值，並使用這些推薦值來提高叢集的資源利用率。

請參閱 [本文件](tutorials/analytics-and-recommendation.md) 了解更多信息。

### QoS 保證
Kubernetes 能夠在同一個節點上啟動多個 Pod，因此當存在資源（例如 cpu）消耗競爭時，部分用戶應用程式可能會受到影響。 為了緩解這種情況，Crane 允許使用者為 Pod 和 QoSEnsurancePolicy 定義優先級，然後檢測中斷並確保高優先級Pod不受資源競爭的影響。

迴避措施：

- **Disable Schedule**：通過設定節點污染和條件來關閉調度
- **Throttle**: 通過壓縮 cgroup 設定來限制低優先級的Pod
- **Evict**: 關閉低優先級的Pod

請參閱 [本文件](tutorials/using-qos-ensurance.md) 了解更多信息。

## 負載感知調度
Kubernetes 的原生調度器只能通過資源請求來調度 Pod，容易造成一系列負載不均的問題。 相比之下，Crane-scheduler 可以從 Prometheus 獲取 kubernetes 節點的實際負載，實現更高效的調度。

請參閱 [本文件](tutorials/scheduling-pods-based-on-actual-node-load.md) 了解更多信息。

## 儲存庫

Crane 由以下元件組成：

- [craned](https://github.com/gocrane/crane/tree/main/cmd/craned) -  crane 主要控制平面。
    - **Predictor** - 根據歷史數據預測資源指標趨勢。
    - **AnalyticsController** - 分析資源並產生相關建議。
    - **RecommendationController** - 推薦 Pod 資源請求和自動縮放器。
    - **ClusterNodePredictionController** - 為節點創建預測器。
    - **EffectiveHPAController** - 用於水平縮放的高效HPA。
    - **EffectiveVPAController** - 用於垂直縮放的高效VPA。
- [metric-adaptor](https://github.com/gocrane/crane/tree/main/cmd/metric-adapter) - 用於驅動擴展的度量服務器。
- [crane-agent](https://github.com/gocrane/crane/tree/main/cmd/crane-agent) - 確保基於異常檢測的關鍵工作負載SLO。
- [gocrane/api](https://github.com/gocrane/api) - 該存儲庫為 Crane 平台定義了組件級 API。
- [gocrane/fadvisor](https://github.com/gocrane/fadvisor) - 從雲端API收集資源價格的財務顧問。
- [gocrane/crane-scheduler](https://github.com/gocrane/crane-scheduler) - 一個 Kubernetes 調度器，可以根據實際節點負載調度pod。
