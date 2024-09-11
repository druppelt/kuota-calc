package calc

import (
	"math"

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

	cpuMin, cpuMax, memoryMin, memoryMax := podResources(&s.Spec.Template.Spec)

	memMin := float64(memoryMin.Value()) * float64(replicas)
	memoryMin.Set(int64(math.Round(memMin)))

	memMax := float64(memoryMax.Value()) * float64(replicas)
	memoryMax.Set(int64(math.Round(memMax)))

	cpuMin.SetMilli(int64(math.Round(float64(cpuMin.MilliValue()) * float64(replicas))))

	cpuMax.SetMilli(int64(math.Round(float64(cpuMax.MilliValue()) * float64(replicas))))

	resourceUsage := ResourceUsage{
		CpuMin:    cpuMin,
		CpuMax:    cpuMax,
		MemoryMin: memoryMin,
		MemoryMax: memoryMax,
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
