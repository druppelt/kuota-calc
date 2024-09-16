package calc

import v1 "k8s.io/api/core/v1"

func pod(pod v1.Pod) *ResourceUsage {
	podResources := calcPodResources(&pod.Spec)

	resourceUsage := ResourceUsage{
		NormalResources:  podResources.Containers,
		RolloutResources: podResources.MaxResources,
		Details: Details{
			Version:     pod.APIVersion,
			Kind:        pod.Kind,
			Name:        pod.Name,
			Strategy:    "",
			Replicas:    0,
			MaxReplicas: 0,
		},
	}

	return &resourceUsage
}
