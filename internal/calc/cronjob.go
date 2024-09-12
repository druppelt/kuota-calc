package calc

import batchV1 "k8s.io/api/batch/v1"

func cronjob(cronjob batchV1.CronJob) *ResourceUsage {
	podResources := podResources(&cronjob.Spec.JobTemplate.Spec.Template.Spec)

	resourceUsage := ResourceUsage{
		resources: *podResources,
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
