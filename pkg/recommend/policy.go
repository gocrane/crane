package recommend

var recommenderPolicy RecommendationPolicy

func init() {
	recommenderPolicy = RecommendationPolicy{
		Spec: RecommendationPolicySpec{
			InspectorPolicy: InspectorPolicy{
				PodAvailableRatio:  0.5,
				PodMinReadySeconds: 0,
				DeploymentMinReplicas: 2,
				StatefulSetMinReplicas: 2,
				WorkloadMinReplicas: 2,
			},
		},
	}

	//todo: initialization policy from file or configmap
}
