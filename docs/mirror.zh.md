# 镜像仓库

## 关于镜像仓库

因为各种网络问题，导致部分地域难以访问GitHub 资源，如GitHub Repo, GitHub Release, GitHub Raw Content `raw.githubusercontent.com`。

为了更好的使用体验，GoCrane 为您额外提供了多个镜像仓库，但具有一定的时延。

## Helm Resources

!!! tips 
    每六小时同步一次上游的最新版本

| Origin                                         | Mirror                                              | Type | Public |
| --------------------------------------------- | --------------------------------------------------------- | ------ | ----- |
| https://gocrane.github.io/helm-charts | https://finops-helm.pkg.coding.net/gocrane/gocrane | Helm | [Public](https://finops.coding.net/public-artifacts/gocrane/gocrane/packages) |
|  https://prometheus-community.github.io/helm-charts  | https://finops-helm.pkg.coding.net/gocrane/prometheus-community    | Helm | [Public](https://finops.coding.net/public-artifacts/gocrane/prometheus-community/packages) |
| https://grafana.github.io/helm-charts      | https://finops-helm.pkg.coding.net/gocrane/grafana      | Helm | [Public](https://finops.coding.net/public-artifacts/gocrane/grafana/packages) |

## Git Resources

!!! tips 
    每天同步一次上游仓库

| Origin                                         | Mirror                                              | Type | Public |
| --------------------------------------------- | --------------------------------------------------------- | ------ | ---- |
| https://github.com/gocrane/crane.git | https://e.coding.net/finops/gocrane/crane.git | Git | [Public](https://finops.coding.net/public/gocrane/crane/git/files) |
| https://github.com/gocrane/helm-charts.git | https://e.coding.net/finops/gocrane/helm-charts.git | Git | [Public](https://finops.coding.net/public/gocrane/helm-charts/git/files) |
| https://github.com/gocrane/api.git | https://e.coding.net/finops/gocrane/api.git | Git | [Public](https://finops.coding.net/public/gocrane/api/git/files) |
| https://github.com/gocrane/crane-scheduler.git | https://e.coding.net/finops/gocrane/crane-scheduler.git | Git | [Public](https://finops.coding.net/public/gocrane/crane-scheduler/git/files) |
| https://github.com/gocrane/fadvisor.git | https://e.coding.net/finops/gocrane/fadvisor.git | Git | [Public](https://finops.coding.net/public/gocrane/fadvisor/git/files) |

## 获取 Coding Git 仓库源文件内容

在这里将为您介绍，如何通过HTTP请求直接获取 Coding Git 仓库中的源文件内容。

### Coding Git 仓库的关键参数

与常规的API请求类似，Coding Git仓库提供了对应的API接口。

下面为您介绍相关的参数。

!!! tips Example "Example"
    以 https://**finops**.coding.net/public/**gocrane**/**helm-charts**/git/files**/main/integration/grafana/override_values.yaml** 作为例子。 [点击访问](https://finops.coding.net/public/gocrane/helm-charts/git/files/main/integration/grafana/override_values.yaml)

| 参数 | 说明 | 例子 |
| ---- | ---- | ---- |
| `team` | 团队名称 | `finops` |
| `project` | 项目名称 | `gocrane` |
| `repo` | Git 仓库名称 | `helm-charts` |
| `branch` | 分支名称 | `main` |
| `file path` | 项目中的文件路径 | `/integration/grafana/override_values.yaml` |

### 构造HTTP请求

根据上面所提到的属性，按照下面的URL构造规则依次填入，即可获得一个可以直接获取源文件内容的URL。

```bash
https://<team>.coding.net/p/<project>/d/<repo>/git/raw/<branch>/<file path>?download=false

https://finops.coding.net/p/gocrane/d/helm-charts/git/raw/main/integration/grafana/override_values.yaml?download=false
```

!!! tips
    尝试以下的命令

```bash
curl https://finops.coding.net/p/gocrane/d/helm-charts/git/raw/main/integration/grafana/override_values.yaml?download=false
```
