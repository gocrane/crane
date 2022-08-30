---
title: "镜像资源"
weight: 90
description: >
  Crane 的镜像资源文档.
---

## 关于镜像资源

因为各种网络问题，导致部分地域难以访问GitHub 资源，如GitHub Repo, GitHub Release, GitHub Raw Content `raw.githubusercontent.com`。

为了更好的使用体验，GoCrane 为您额外提供了多个镜像仓库，但具有一定的时延。

## Image Registry

GoCrane提供了一种友好的方式来使用镜像进行部署和测试。

GoCrane使用CI(GitHub Action)进行镜像构建。

### Platforms

GoCrane 现在支持 linux/amd64 和 linux/arm64。

GoCrane 也在关注ARM用户，譬如 apple m1/m2。

### Repo

由于网络原因，GoCrane 将同时将镜像推送到三个镜像仓库。

!!! tips
点击链接，查阅更多

- [DockerHub](https://hub.docker.com/u/gocrane)
- [Coding](https://finops.coding.net/public-artifacts/gocrane/crane/packages)
- [GitHub Container Registry](https://github.com/orgs/gocrane/packages?repo_name=crane)

如果你在中国，我们推荐使用 Coding。该仓库的速度比其他两个快。

如果你在中国境外，我们推荐你使用 DockHub以及GHCR。如果你使用Coding，速度上面不太理想。

### Build logic

- 每个分支

  你可以使用该分支镜像尝试最新的特性。此外，我们依旧保存较早之前的镜像。

- 每个pull request

  当你发起一个pull request到crane repo的时候，将会自动触发CI任务来构建对应的镜像。CI会将最后的镜像结果通过评论的方式附加在对应的pull request中。

### How to use the images?

这里将使用 main 分支作为例子。
Git commit hash 为 abc123。

#### Base on the branch name

!!! tips
branch name的镜像一直指向最新的Git commit。当你想尝试新特性的时候，不要忘记重新拉取镜像。

=== "DockerHub"
```bash
docker pull gocrane/craned:main
```

=== "Coding"
```bash
docker pull finops-docker.pkg.coding.net/gocrane/crane/craned:main
```

=== "GitHub Container Registry"
```bash
docker pull ghcr.io/gocrane/crane/craned:main
```

#### Base on the branch name and the specific commit hash

=== "DockerHub"
```bash
docker pull gocrane/craned:main-abc123
```

=== "Coding"
```bash
docker pull finops-docker.pkg.coding.net/gocrane/crane/craned:main-abc123
```

=== "GitHub Container Registry"
```bash
docker pull ghcr.io/gocrane/crane/craned:main-abc123
```

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

!!! warning
Now Coding is not support to fetch raw contents directly. You must be get token first.

### Coding
| Origin                                         | Mirror                                              | Type | Public |
| --------------------------------------------- | --------------------------------------------------------- | ------ | ---- |
| https://github.com/gocrane/crane.git | https://e.coding.net/finops/gocrane/crane.git | Git | [Public](https://finops.coding.net/public/gocrane/crane/git/files) |
| https://github.com/gocrane/helm-charts.git | https://e.coding.net/finops/gocrane/helm-charts.git | Git | [Public](https://finops.coding.net/public/gocrane/helm-charts/git/files) |
| https://github.com/gocrane/api.git | https://e.coding.net/finops/gocrane/api.git | Git | [Public](https://finops.coding.net/public/gocrane/api/git/files) |
| https://github.com/gocrane/crane-scheduler.git | https://e.coding.net/finops/gocrane/crane-scheduler.git | Git | [Public](https://finops.coding.net/public/gocrane/crane-scheduler/git/files) |
| https://github.com/gocrane/fadvisor.git | https://e.coding.net/finops/gocrane/fadvisor.git | Git | [Public](https://finops.coding.net/public/gocrane/fadvisor/git/files) |

### Gitee

| Origin                                         | Mirror                                              | Type | Public |
| --------------------------------------------- | --------------------------------------------------------- | ------ | ---- |
| https://github.com/gocrane/crane.git | https://gitee.com/finops/crane | Git | [Public](https://gitee.com/finops/crane) |
| https://github.com/gocrane/helm-charts.git | https://gitee.com/finops/helm-charts | Git | [Public](https://gitee.com/finops/helm-charts) |

## 获取 Coding Git 仓库源文件内容

!!! warning
Now Coding is not support to fetch raw contents directly. You must be get token first.

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
