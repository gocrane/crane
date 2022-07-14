# Crane 雲端資源分析與效益

[![Go Report Card](https://goreportcard.com/badge/github.com/gocrane/crane)](https://goreportcard.com/report/github.com/gocrane/crane)
[![GoDoc](https://godoc.org/github.com/gocrane/crane?status.svg)](https://godoc.org/github.com/gocrane/crane)
[![License](https://img.shields.io/github/license/gocrane/crane)](https://www.apache.org/licenses/LICENSE-2.0.html)
![GoVersion](https://img.shields.io/github/go-mod/go-version/gocrane/crane)

<img alt="Crane logo" height="100" src="docs/images/crane.svg" title="Crane" width="200"/>

---

Crane (FinOps Crane) 是一個雲端原生開源計畫並可管理Kubernetes叢集上的雲端資源, 此計畫受到FinOps的概念啟發。

## 介紹

Crane目標是為了協助當Kubernetes使用者透過大量的雲端功能時，能夠節省雲端資源使用量的一個的專案。

- 基於監控資料的**時間序列預測**
- **資源使用率與成本的可視化**
- **資源使用率與成本的優化** 包含:
  - R2 資源的重新分配(Resource Re-allocation)
  - R3 請求和副本的建議(Request & Replicas Recommendation)
  - 高效率的Pod自動彈性化 (高效率的垂直與橫向的Pod自動彈性化)
  - 成本最佳化
- 基於Pod的優先級**增強QoS**
- **負載感知調度**

<img alt="Crane Overview" height="550" src="docs/images/crane-overview.png" width="800"/>

## Getting Started (開始使用)

- [介紹](https://docs.gocrane.io)
- [安裝](https://docs.gocrane.io/dev/installation/)
- [使用教學](https://docs.gocrane.io/dev/tutorials/using-effective-hpa-to-scaling-with-effectiveness/)

## 文件

完整的文件可在Crane的官方網站上取得 [Crane website](https://docs.gocrane.io)。

## 社群

- 微信群(中文):加入群內後回覆Crane 機器人將會加入你至微信群。

<img alt="Wechat" src="docs/images/wechat.jpeg" title="Wechat" width="200"/>

- 兩周一次的社群會議(APAC, Chinese)
  - [會議連結](https://meeting.tencent.com/dm/SjY20wCJHy5F)
  - [會議事項](https://doc.weixin.qq.com/doc/w3_AHMAlwa_AFU7PT58rVhTFKXV0maR6?scode=AJEAIQdfAAo0gvbrCIAHMAlwa_AFU)
  - [會議影片紀錄](https://www.wolai.com/33xC4HB1JXCCH1x8umfioS)

## 產品路線圖

請透過此[文件](./docs/roadmaps/roadmap-1h-2022.md)查看更多資訊。

## 貢獻

我們歡迎貢獻者加入我們的crane計畫 若想知道如何為此計畫進行貢獻 請查看[貢獻](./CONTRIBUTING.md)獲取相關資訊。

## 行為準則

Crane adopts [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).
