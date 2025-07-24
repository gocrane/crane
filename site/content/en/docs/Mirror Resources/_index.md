---
title: "Mirror Resources"
linkTitle: "Mirror Resources"
weight: 90
description: >
  List all Mirror Resources in Crane.
---

## About mirror resources

Because of various network issues, it is difficult to access GitHub resources such as GitHub Repo, GitHub Release, GitHub Raw Content `raw.githubusercontent.com` in some regions.

For a better experience, GoCrane offers several additional mirror repositories for you, but with some latency.

## Image Registry

GoCrane provides a friendly way to use images to deploy and test.

GoCrane builds images based on the CI(GitHub Action).

### Platforms

GoCrane now supports linux/amd64 and linux/arm64.

GoCrane still cares about arm users, like apple m1/m2.

### Repo
Because of the network problems, GoCrane pushes the images to three different repo at the same time.

!!! tips
Click these links to see details.
- [DockerHub](https://hub.docker.com/u/gocrane)
- [Coding](https://finops.coding.net/public-artifacts/gocrane/crane/packages)
- [GitHub Container Registry](https://github.com/orgs/gocrane/packages?repo_name=crane)

If you locate in China, we recommend using the Coding repo. It's fast than other registry repo.

If you locate outside of China, we recommend using DockerHub and GitHub Container Registry. However, if you use Coding, the Registry may be slow.

### Build logic

- Each branch

  You can try the new features based on the branch images. In addition, we still reserve the early images.

- Each pull request

  When you make a pull request to the crane repo, that will trigger CI to build images. In addition, a comment will include image info to the pull request when CI completes.

### How to use the images?

Here use the main branch as an example.
The git commit hash is abc123.

#### Base on the branch name

!!! tips
The branch name still points to the last commit. Don't forget to re-pull the images when you want to try the new features.

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
Sync the latest version of upstream every six hours

| Origin                                         | Mirror                                              | Type | Public |
| --------------------------------------------- | --------------------------------------------------------- | ------ | ----- |
| https://gocrane.github.io/helm-charts | https://finops-helm.pkg.coding.net/gocrane/gocrane | Helm | [Public](https://finops.coding.net/public-artifacts/gocrane/gocrane/packages) |
|  https://prometheus-community.github.io/helm-charts  | https://finops-helm.pkg.coding.net/gocrane/prometheus-community    | Helm | [Public](https://finops.coding.net/public-artifacts/gocrane/prometheus-community/packages) |
| https://grafana.github.io/helm-charts      | https://finops-helm.pkg.coding.net/gocrane/grafana      | Helm | [Public](https://finops.coding.net/public-artifacts/gocrane/grafana/packages) |

## Git Resources

!!! tips
Sync upstream repository every day

### Coding

!!! warning
Now Coding is not support to fetch raw contents directly. You must be get token first.

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
| https://github.com/gocrane/crane-scheduler.git | https://gitee.com/finops/crane-scheduler | Git | [Public](https://gitee.com/finops/crane-scheduler) |

## Get the raw file contents of the Coding repo

!!! warning
Now Coding is not support to fetch raw contents directly. You must be get token first.

Here you'll find out how to get the contents of a source file directly from the Coding Git repository via an HTTP request.

### Coding Git Repo - Key Params

Similar to regular API requests, the Coding Git repository provides a corresponding API interface.

The following is an overview of the related parameters.

!!! tips Example "Example"
Using https://**finops**.coding.net/public/**gocrane**/**helm-charts**/git/files**/main/integration/grafana/override_values.yaml** as an example. [Click Here](https://finops.coding.net/public/gocrane/helm-charts/git/files/main/integration/grafana/override_values.yaml)

| Params | Description | example |
| ---- | ---- | ---- |
| `team` | Name of the team | `finops` |
| `project` | Name of the project | `gocrane` |
| `repo` | Name of the Git Repo | `helm-charts` |
| `branch` | Name of the branch | `main` |
| `file path` | The path to the file in the repo | `/integration/grafana/override_values.yaml` |

### Constructing HTTP requests

By filling in the following URL construction rules according to the properties mentioned above, you can obtain a URL that can directly access the content of the source file.

```bash
https://<team>.coding.net/p/<project>/d/<repo>/git/raw/<branch>/<file path>?download=false

https://finops.coding.net/p/gocrane/d/helm-charts/git/raw/main/integration/grafana/override_values.yaml?download=false
```

!!! tips
Try this command.

```bash
curl https://finops.coding.net/p/gocrane/d/helm-charts/git/raw/main/integration/grafana/override_values.yaml?download=false
```
