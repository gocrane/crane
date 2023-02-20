#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

function help()
{
  cat  <<EOF

The crane local environment setup

Usage: local-env-setup.sh <[Options]>
Options:
         -h   --help           help for setup
         -m   --mirror         setup crane from helm mirror repo
EOF

}

FROM_MIRROR=false

while [ $# -gt 0 ]
do
    case $1 in
    -h|--help) help ; exit 1;;
    -m|--mirror) FROM_MIRROR=true ;;
    (-*) echo "$0: error - unrecognized option $1" 1>&2; help; exit 1;;
    (*) break;;
    esac
    shift
done


CRANE_KUBECONFIG="${HOME}/.kube/config_crane"
CRANE_CLUSTER_NAME="crane"

PROMETHEUS_HELM_NAME="prometheus-community"
PROMETHEUS_HELM_URL="https://prometheus-community.github.io/helm-charts"
PROMETHEUS_VALUE_URL="https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/prometheus/override_values.yaml"
GRAFANA_HELM_NAME="grafana"
GRAFANA_HELM_URL="https://grafana.github.io/helm-charts"
GRAFANA_VALUE_URL="https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/grafana/override_values.yaml"
CRANE_HELM_NAME="crane"
CRANE_HELM_URL="https://gocrane.github.io/helm-charts"

# check if setup is from mirror repo

if [ "$FROM_MIRROR" = true ]; then
  PROMETHEUS_HELM_NAME="prometheus-community-gocrane"
  PROMETHEUS_HELM_URL="https://finops-helm.pkg.coding.net/gocrane/prometheus-community"
  PROMETHEUS_VALUE_URL="https://gitee.com/finops/helm-charts/raw/main/integration/prometheus/override_values.yaml"
  GRAFANA_HELM_NAME="grafana-gocrane"
  GRAFANA_HELM_URL="https://finops-helm.pkg.coding.net/gocrane/grafana"
  GRAFANA_VALUE_URL="https://gitee.com/finops/helm-charts/raw/main/integration/grafana/override_values.yaml"
  CRANE_HELM_NAME="crane-mirror"
  CRANE_HELM_URL="https://finops-helm.pkg.coding.net/gocrane/gocrane"
fi

echo "Step1: Create local cluster: " ${CRANE_KUBECONFIG}
kind delete cluster --name="${CRANE_CLUSTER_NAME}" 2>&1
kind create cluster --kubeconfig "${CRANE_KUBECONFIG}" --name "${CRANE_CLUSTER_NAME}" --image kindest/node:v1.21.1
export KUBECONFIG="${CRANE_KUBECONFIG}"
echo "Step1: Create local cluster finished."

echo "Step2: Installing Prometheus "
helm repo add ${PROMETHEUS_HELM_NAME} ${PROMETHEUS_HELM_URL}
helm install prometheus -n crane-system \
                        --set prometheus-pushgateway.enabled=false \
                        --set alertmanager.enabled=false \
                        --set server.persistentVolume.enabled=false \
                        -f ${PROMETHEUS_VALUE_URL} \
                        --create-namespace  ${PROMETHEUS_HELM_NAME}/prometheus
echo "Step2: Installing Prometheus finished."

echo "Step3: Installing Grafana "
helm repo add ${GRAFANA_HELM_NAME} ${GRAFANA_HELM_URL}
helm install grafana \
             -f ${GRAFANA_VALUE_URL} \
             -n crane-system \
             --create-namespace ${GRAFANA_HELM_NAME}/grafana
echo "Step3: Installing Grafana finished."

echo "Step4: Installing Crane "
helm repo add ${CRANE_HELM_NAME} ${CRANE_HELM_URL}
helm repo update
helm install crane -n crane-system --set craneAgent.enable=false --create-namespace ${CRANE_HELM_NAME}/crane
helm install fadvisor -n crane-system --create-namespace ${CRANE_HELM_NAME}/fadvisor
echo "Step4: Installing Crane finished."

kubectl get deploy -n crane-system
echo "Please wait for all pods ready"
echo "After all pods ready, Get the Crane Dashboard URL to visit by running these commands in the same shell:"
echo "    export KUBECONFIG=${HOME}/.kube/config_crane"
echo "    kubectl -n crane-system port-forward service/craned 9090:9090"

