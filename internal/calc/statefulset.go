package calc

import (
	appsv1 "k8s.io/api/apps/v1"
)

// calculates the cpu/memory resources a single statefulset needs. Replicas are taken into account.
func statefulSet(s appsv1.StatefulSet) *ResourceUsage {
	var (
		replicas int32
	)

	// https://github.com/kubernetes/api/blob/v0.18.4/apps/v1/types.go#L117
	if s.Spec.Replicas != nil {
		replicas = *s.Spec.Replicas
	} else {
		replicas = 1
	}

	podResources := podResources(&s.Spec.Template.Spec)
	newResources := (*podResources).Mul(float64(replicas))

	resourceUsage := ResourceUsage{
		resources: newResources,
		Details: Details{
			Version:     s.APIVersion,
			Kind:        s.Kind,
			Name:        s.Name,
			Replicas:    replicas,
			Strategy:    string(s.Spec.UpdateStrategy.Type),
			MaxReplicas: replicas,
		},
	}

	return &resourceUsage
}
