# Developer Guide #
- Thanks for taking the time to contribute to crane!

- Please review and follow the [Code of Conduct](https://github.com/gocrane/crane/blob/main/CODE_OF_CONDUCT.md)


- Welcome to the Crane Developer Guide
- Crane (FinOps Crane) is a cloud native open source project which manages cloud resources on Kubernetes stack, it is inspired by FinOps concepts.

In this guide I will show you step by step how you can contribute your learnings here.

### Steps Involved
- You need a github account and you need the fork the this [repository](https://github.com/gocrane/crane.git). 

- After your have forked the repo you will see `<username>/crane` under your account.
  
- Now you need to clone the repo into your local system. For that open the terminal and clone. 
```
git clone https://github.com/<username>/crane
```
- then you need go inside the directory 
```cd crane ```
- You need to set upstream url, run this command to do so:
```

git remote add upstream https://github.com/gocrane/crane.git
```

- Run these commands to add a different branch:
   - You can have anything as <branch_name> , but by convention it should indicate what you are working on
      -example : <branch_name> can be **add-folder**, when you are adding your folder. 
```
git branch <branch_name>

git checkout <branch_name>
``` 

- To push your changes to github, Run the following commands.
```
git add .
```
- Write a good commit messages.
  - A simple but effective convention to follow for commits is the “problem / solution” pattern. It looks like this:

```
<Subject>

Problem: <Statement of problem>

Solution: <Statement of solution>
```

```
git commit -m "problem/solution"

git push origin <branch_name>
```

- Go to your github forked repo, You will see an option to "Compare and Pull request".
- Click on that and Then You will see an option to "Create pull request". Click on that.
- that's it you have made your pull request.
- After your request is accepted, You will see the folder of your name on the repository.
 
### Setting up local environments for Kubernetes

- Basic Setup

- To set up the development environment for the developers, the machine needs a working Kubernetes node, with other tools installed. The developer setup requires:

- Minikube
- kubectl
- Helm
- Draft, Skaffold, or Garden.io
- Docker (optional)

- To automate local deployment, we could use many tools available like Draft, Skaffold, or Garden.io. We prefer using Helm  for local development.

### Helm Installation¶
Please refer to Helm's [documentation](https://helm.sh/docs/intro/install/) for installation.


### Steps to Setup
- Crane use prometheus to be the default metric provider.
- Using following command to install prometheus components: prometheus-server, node-exporter, kube-state-metrics.

```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus -n crane-system \
                        --set pushgateway.enabled=false \
                        --set alertmanager.enabled=false \
                        --set server.persistentVolume.enabled=false \
                        -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/prometheus/override_values.yaml \
                        --create-namespace  prometheus-community/prometheus
```
- Fadvisor use grafana to present cost estimates. Using following command to install a grafana.

```
helm repo add grafana https://grafana.github.io/helm-charts
helm install grafana \
             -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/grafana/override_values.yaml \
             -n crane-system \
             --create-namespace grafana/grafana

```
- Deploying Crane and Fadvisor¶

```
helm repo add crane https://gocrane.github.io/helm-charts
helm install crane -n crane-system --create-namespace crane/crane
helm install fadvisor -n crane-system --create-namespace crane/fadvisor
```

- Deploying Crane-scheduler(optional)¶

```
helm install scheduler -n crane-system --create-namespace crane/scheduler
```

- Verify Installation¶
  - Check deployments are all available by running:

```
kubectl get deploy -n crane-system
```

- The output is similar to:

```
NAME                                             READY   STATUS    RESTARTS   AGE
crane-agent-8h7df                                1/1     Running   0          119m
crane-agent-8qf5n                                1/1     Running   0          119m
crane-agent-h9h5d                                1/1     Running   0          119m
craned-5c69c684d8-dxmhw                          2/2     Running   0          20m
grafana-7fddd867b4-kdxv2                         1/1     Running   0          41m
metric-adapter-94b6f75b-k8h7z                    1/1     Running   0          119m
prometheus-kube-state-metrics-6dbc9cd6c9-dfmkw   1/1     Running   0          45m
prometheus-node-exporter-bfv74                   1/1     Running   0          45m
prometheus-node-exporter-s6zps                   1/1     Running   0          45m
prometheus-node-exporter-x5rnm                   1/1     Running   0          45m
prometheus-server-5966b646fd-g9vxl               2/2     Running   0          45m
```

you can see this to learn more.

- Customize Installation¶
  - Deploy `Crane` by apply YAML declaration.

```
git clone https://github.com/gocrane/crane.git
CRANE_LATEST_VERSION=$(curl -s https://api.github.com/repos/gocrane/crane/releases/latest | grep -oP '"tag_name": "\K(.*)(?=")')
git checkout $CRANE_LATEST_VERSION
kubectl apply -f deploy/manifests 
kubectl apply -f deploy/craned 
kubectl apply -f deploy/metric-adapter
```

The following command will configure prometheus http address for crane if you want to customize it. Specify CUSTOMIZE_PROMETHEUS if you have existing prometheus server.

```
export CUSTOMIZE_PROMETHEUS=
if [ $CUSTOMIZE_PROMETHEUS ]; then sed -i '' "s/http:\/\/prometheus-server.crane-system.svc.cluster.local:8080/${CUSTOMIZE_PROMETHEUS}/" deploy/craned/deployment.yaml ; fi
```

- Get your Kubernetes Cost Report¶
  - Get the Grafana URL to visit by running these commands in the same shell:

```
export POD_NAME=$(kubectl get pods --namespace crane-system -l "app.kubernetes.io/name=grafana,app.kubernetes.io/instance=grafana" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace crane-system port-forward $POD_NAME 3000
```

  
 
