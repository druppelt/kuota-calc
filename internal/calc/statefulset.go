package calc

import (
	"errors"
	"math"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// calculates the cpu/memory resources a single statefulset needs. Replicas are taken into account.
func statefulSet(s appsv1.StatefulSet) (*ResourceUsage, error) {
	var (
		replicas       int32
		maxUnavailable int32
	)

	strategy := s.Spec.UpdateStrategy

	// https://github.com/kubernetes/api/blob/v0.18.4/apps/v1/types.go#L117
	if s.Spec.Replicas != nil {
		replicas = *s.Spec.Replicas
	} else {
		replicas = 1
	}

	// https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#update-strategies
	switch strategy.Type {
	case appsv1.OnDeleteStatefulSetStrategyType:
		// OnDelete doesn't do anything until you kill pods, which it then replaces with the newer ones.
		// The most expensive case would be killing all pods at once, with the init containers being more expensive than the normal container.
		maxUnavailable = replicas
	case "":
		// RollingUpdate is the default and can be an empty string. If so, set the defaults and continue calculation.
		defaultMaxUnavailable := intstr.FromInt32(1)
		strategy = appsv1.StatefulSetUpdateStrategy{
			Type: appsv1.RollingUpdateStatefulSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateStatefulSetStrategy{
				MaxUnavailable: &defaultMaxUnavailable,
			},
		}

		fallthrough
	case appsv1.RollingUpdateStatefulSetStrategyType:
		// RollingUpdate updates each Pod one at a time. It waits until an updated Pod is Running and Ready before continuing with the next pod.
		// There is an alpha feature to support rollout of multiple pods at once with `.spec.updateStrategy.rollingUpdate.maxUnavailable`
		var maxUnavailableValue intstr.IntOrString

		if strategy.RollingUpdate == nil {
			maxUnavailableValue = intstr.FromInt32(1)
		} else {
			maxUnavailableValue = *strategy.RollingUpdate.MaxUnavailable
		}

		// docs say, that the absolute number is calculated by rounding up.
		maxUnavailableInt, err := intstr.GetScaledValueFromIntOrPercent(&maxUnavailableValue, int(replicas), true)
		if err != nil {
			return nil, err
		}

		if maxUnavailableInt < math.MinInt32 || maxUnavailableInt > math.MaxInt32 {
			return nil, errors.New("maxUnavailableInt out of int32 boundaries")
		}

		maxUnavailable = int32(maxUnavailableInt)
	}

	podResources := calcPodResources(&s.Spec.Template.Spec)
	rolloutResources := podResources.Containers.MulInt32(replicas - maxUnavailable).Add(podResources.MaxResources.MulInt32(maxUnavailable))
	normalResources := podResources.Containers.MulInt32(replicas)

	resourceUsage := ResourceUsage{
		NormalResources:  normalResources,
		RolloutResources: rolloutResources,
		Details: Details{
			Version:     s.APIVersion,
			Kind:        s.Kind,
			Name:        s.Name,
			Replicas:    replicas,
			Strategy:    string(strategy.Type),
			MaxReplicas: replicas,
		},
	}

	return &resourceUsage, nil
}
