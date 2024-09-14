package calc

import (
	appsv1 "k8s.io/api/apps/v1"
)

func daemonSet(dSet appsv1.DaemonSet) *ResourceUsage {
	podResources := calcPodResources(&dSet.Spec.Template.Spec).MaxResources

	resourceUsage := ResourceUsage{
		Resources: podResources,
		Details: Details{
			Version:     dSet.APIVersion,
			Kind:        dSet.Kind,
			Name:        dSet.Name,
			Strategy:    "",
			Replicas:    1,
			MaxReplicas: 1,
		},
	}

	return &resourceUsage
}
