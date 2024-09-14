package calc

import batchV1 "k8s.io/api/batch/v1"

func cronjob(cronjob batchV1.CronJob) *ResourceUsage {
	podResources := calcPodResources(&cronjob.Spec.JobTemplate.Spec.Template.Spec).MaxResources

	resourceUsage := ResourceUsage{
		Resources: podResources,
		Details: Details{
			Version:     cronjob.APIVersion,
			Kind:        cronjob.Kind,
			Name:        cronjob.Name,
			Strategy:    "",
			Replicas:    0,
			MaxReplicas: 0,
		},
	}

	return &resourceUsage
}
