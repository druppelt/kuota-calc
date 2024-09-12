package calc

import v1 "k8s.io/api/core/v1"

func pod(pod v1.Pod) *ResourceUsage {
	podResources := podResources(&pod.Spec)

	resourceUsage := ResourceUsage{
		resources: *podResources,
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
