# Mirror Repo


## About mirror repo

Because of various network issues, it is difficult to access GitHub resources such as GitHub Repo, GitHub Release, GitHub Raw Content `raw.githubusercontent.com` in some regions.

For a better experience, GoCrane offers several additional mirror repositories for you, but with some latency.

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

| Origin                                         | Mirror                                              | Type | Public |
| --------------------------------------------- | --------------------------------------------------------- | ------ | ---- |
| https://github.com/gocrane/crane.git | https://e.coding.net/finops/gocrane/crane.git | Git | [Public](https://finops.coding.net/public/gocrane/crane/git/files) |
| https://github.com/gocrane/helm-charts.git | https://e.coding.net/finops/gocrane/helm-charts.git | Git | [Public](https://finops.coding.net/public/gocrane/helm-charts/git/files) |
| https://github.com/gocrane/api.git | https://e.coding.net/finops/gocrane/api.git | Git | [Public](https://finops.coding.net/public/gocrane/api/git/files) |
| https://github.com/gocrane/crane-scheduler.git | https://e.coding.net/finops/gocrane/crane-scheduler.git | Git | [Public](https://finops.coding.net/public/gocrane/crane-scheduler/git/files) |
| https://github.com/gocrane/fadvisor.git | https://e.coding.net/finops/gocrane/fadvisor.git | Git | [Public](https://finops.coding.net/public/gocrane/fadvisor/git/files) |

## Get the raw file contents of the Coding repo

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
