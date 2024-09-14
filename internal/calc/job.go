package calc

import batchV1 "k8s.io/api/batch/v1"

func job(job batchV1.Job) *ResourceUsage {
	podResources := calcPodResources(&job.Spec.Template.Spec).MaxResources

	resourceUsage := ResourceUsage{
		Resources: podResources,
		Details: Details{
			Version:     job.APIVersion,
			Kind:        job.Kind,
			Name:        job.Name,
			Strategy:    "",
			Replicas:    0,
			MaxReplicas: 0,
		},
	}

	return &resourceUsage
}
