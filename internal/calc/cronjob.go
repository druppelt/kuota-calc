package calc

import batchV1 "k8s.io/api/batch/v1"

func cronjob(cronjob batchV1.CronJob) *ResourceUsage {
	podResources := calcPodResources(&cronjob.Spec.JobTemplate.Spec.Template.Spec)

	resourceUsage := ResourceUsage{
		// TODO should jobs always be considered with their rollout resources?
		NormalResources:  podResources.Containers,
		RolloutResources: podResources.MaxResources,
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
