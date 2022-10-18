#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

CRANE_KUBECONFIG="${HOME}/.kube/config_crane"
CRANE_CLUSTER_NAME="crane"

echo "Step1: Create local cluster: " ${CRANE_KUBECONFIG}
kind delete cluster --name="${CRANE_CLUSTER_NAME}" 2>&1
kind create cluster --kubeconfig "${CRANE_KUBECONFIG}" --name "${CRANE_CLUSTER_NAME}" --image kindest/node:v1.21.1
export KUBECONFIG="${CRANE_KUBECONFIG}"
echo "Step1: Create local cluster finished."

echo "Step2: Installing Prometheus "
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus -n crane-system \
                        --set pushgateway.enabled=false \
                        --set alertmanager.enabled=false \
                        --set server.persistentVolume.enabled=false \
                        -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/prometheus/override_values.yaml \
                        --create-namespace  prometheus-community/prometheus
echo "Step2: Installing Prometheus finished."

echo "Step3: Installing Grafana "
helm repo add grafana https://grafana.github.io/helm-charts
helm install grafana \
             -f https://raw.githubusercontent.com/gocrane/helm-charts/main/integration/grafana/override_values.yaml \
             -n crane-system \
             --create-namespace grafana/grafana
echo "Step3: Installing Grafana finished."

echo "Step4: Installing Crane "
helm repo add crane https://gocrane.github.io/helm-charts
helm repo update
helm install crane -n crane-system --set craneAgent.enable=false --create-namespace crane/crane
helm install fadvisor -n crane-system --create-namespace crane/fadvisor
echo "Step4: Installing Crane finished."

kubectl get deploy -n crane-system
echo "Please wait for all pods ready"
echo "After all pods ready, Get the Crane Dashboard URL to visit by running these commands in the same shell:"
echo "    export KUBECONFIG=${HOME}/.kube/config_crane"
echo "    kubectl -n crane-system port-forward service/craned 9090:9090"

