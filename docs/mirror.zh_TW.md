# 鏡像資源

## 關於鏡像資源

因為各種網絡問題，導致部分地域難以訪問GitHub 資源，如GitHub Repo, GitHub Release, GitHub Raw Content `raw.githubusercontent.com`。

為了更好的使用體驗，GoCrane 為您額外提供了多個鏡像倉庫，但具有一定的時延。

## Helm Resources

!!! tips 
    每六小時同步一次上游的最新版本

| Origin                                         | Mirror                                              | Type | Public |
| --------------------------------------------- | --------------------------------------------------------- | ------ | ----- |
| https://gocrane.github.io/helm-charts | https://finops-helm.pkg.coding.net/gocrane/gocrane | Helm | [Public](https://finops.coding.net/public-artifacts/gocrane/gocrane/packages) |
|  https://prometheus-community.github.io/helm-charts  | https://finops-helm.pkg.coding.net/gocrane/prometheus-community    | Helm | [Public](https://finops.coding.net/public-artifacts/gocrane/prometheus-community/packages) |
| https://grafana.github.io/helm-charts      | https://finops-helm.pkg.coding.net/gocrane/grafana      | Helm | [Public](https://finops.coding.net/public-artifacts/gocrane/grafana/packages) |

## Git Resources

!!! tips 
    每天同步一次上游倉庫

| Origin                                         | Mirror                                              | Type | Public |
| --------------------------------------------- | --------------------------------------------------------- | ------ | ---- |
| https://github.com/gocrane/crane.git | https://e.coding.net/finops/gocrane/crane.git | Git | [Public](https://finops.coding.net/public/gocrane/crane/git/files) |
| https://github.com/gocrane/helm-charts.git | https://e.coding.net/finops/gocrane/helm-charts.git | Git | [Public](https://finops.coding.net/public/gocrane/helm-charts/git/files) |
| https://github.com/gocrane/api.git | https://e.coding.net/finops/gocrane/api.git | Git | [Public](https://finops.coding.net/public/gocrane/api/git/files) |
| https://github.com/gocrane/crane-scheduler.git | https://e.coding.net/finops/gocrane/crane-scheduler.git | Git | [Public](https://finops.coding.net/public/gocrane/crane-scheduler/git/files) |
| https://github.com/gocrane/fadvisor.git | https://e.coding.net/finops/gocrane/fadvisor.git | Git | [Public](https://finops.coding.net/public/gocrane/fadvisor/git/files) |

## 獲取 Coding Git 倉庫源文件內容

在這裡將為您介紹，如何通過HTTP請求直接獲取 Coding Git 倉庫中的源文件內容。

### Coding Git 倉庫的關鍵參數

與常規的API請求類似，Coding Git倉庫提供了對應的API接口。

下面為您介紹相關的參數。

!!! tips Example "Example"
    以 https://**finops**.coding.net/public/**gocrane**/**helm-charts**/git/files**/main/integration/grafana/override_values.yaml** 作為例子。 [點擊訪問](https://finops.coding.net/public/gocrane/helm-charts/git/files/main/integration/grafana/override_values.yaml)

| 參數 | 說明 | 例子 |
| ---- | ---- | ---- |
| `team` | 團隊名稱 | `finops` |
| `project` | 項目名稱 | `gocrane` |
| `repo` | Git 倉庫名稱 | `helm-charts` |
| `branch` | 分支名稱 | `main` |
| `file path` | 項目中的文件路徑 | `/integration/grafana/override_values.yaml` |

### 構造HTTP請求

根據上面所提到的屬性，按照下面的URL構造規則依次填入，即可獲得一個可以直接獲取源文件內容的URL。

```bash
https://<team>.coding.net/p/<project>/d/<repo>/git/raw/<branch>/<file path>?download=false

https://finops.coding.net/p/gocrane/d/helm-charts/git/raw/main/integration/grafana/override_values.yaml?download=false
```

!!! tips
    嘗試以下的命令

```bash
curl https://finops.coding.net/p/gocrane/d/helm-charts/git/raw/main/integration/grafana/override_values.yaml?download=false
```
