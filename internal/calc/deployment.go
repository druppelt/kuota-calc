package calc

import (
	"errors"
	"fmt"
	"math"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// calculates the cpu/memory resources a single deployment needs. Replicas and the deployment
// strategy are taken into account.
func deployment(deployment appsv1.Deployment) (*ResourceUsage, error) { //nolint:funlen // disable function length linting
	var (
		maxUnavailable      int32 // max amount of unavailable pods during a deployment
		maxSurge            int32 // max amount of pods that are allowed in addition to replicas during deployment
		maxNonReadyPodCount int32 // max pods that are not ready during deployment,
		//  so either running init containers or already running normal containers,
		//  but probes haven't succeeded yet
	)

	replicas := deployment.Spec.Replicas
	strategy := deployment.Spec.Strategy

	if *replicas == 0 {
		return &ResourceUsage{
			Resources: Resources{},
			Details: Details{
				Version:     deployment.APIVersion,
				Kind:        deployment.Kind,
				Name:        deployment.Name,
				Replicas:    *replicas,
				MaxReplicas: *replicas,
				Strategy:    string(strategy.Type),
			},
		}, nil
	}

	switch strategy.Type {
	case appsv1.RecreateDeploymentStrategyType:
		// kill all existing pods, then recreate new ones at once -> no overhead on recreate
		maxNonReadyPodCount = *replicas
		maxUnavailable = *replicas
		maxSurge = 0
	case "":
		// RollingUpdate is the default and can be an empty string. If so, set the defaults
		// (https://pkg.go.dev/k8s.io/api/apps/v1?tab=doc#RollingUpdateDeployment) and continue calculation.
		defaults := intstr.FromString("25%")
		strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxUnavailable: &defaults,
				MaxSurge:       &defaults,
			},
		}

		fallthrough
	case appsv1.RollingUpdateDeploymentStrategyType:
		// Documentation: https://pkg.go.dev/k8s.io/api/apps/v1?tab=doc#RollingUpdateDeployment
		// all default values are set as stated in the docs
		var (
			maxUnavailableValue intstr.IntOrString
			maxSurgeValue       intstr.IntOrString
		)

		// can be nil, if so apply default value
		if strategy.RollingUpdate == nil {
			maxUnavailableValue = intstr.FromString("25%")
			maxSurgeValue = intstr.FromString("25%")
		} else {
			maxUnavailableValue = *strategy.RollingUpdate.MaxUnavailable
			maxSurgeValue = *strategy.RollingUpdate.MaxSurge
		}

		// docs say, that the absolute number is calculated by rounding down.
		maxUnavailableInt, err := intstr.GetScaledValueFromIntOrPercent(&maxUnavailableValue, int(*replicas), false)
		if err != nil {
			return nil, err
		}

		if maxUnavailableInt < math.MinInt32 || maxUnavailableInt > math.MaxInt32 {
			return nil, errors.New("maxUnavailableInt out of int32 boundaries")
		}

		maxUnavailable = int32(maxUnavailableInt)

		// docs say, absolute number is calculated by rounding up.
		maxSurgeInt, err := intstr.GetScaledValueFromIntOrPercent(&maxSurgeValue, int(*replicas), true)
		if err != nil {
			return nil, err
		}

		if maxSurgeInt < math.MinInt32 || maxSurgeInt > math.MaxInt32 {
			return nil, errors.New("maxSurgeInt out of int32 boundaries")
		}

		maxSurge = int32(maxSurgeInt)

		// maxNonReadyPodCount is the max number of pods potentially in init phase during a deployment
		maxNonReadyPodCount = maxSurge + maxUnavailable
	default:
		return nil, fmt.Errorf("deployment: %s deployment strategy %q is unknown", deployment.Name, strategy.Type)
	}

	podResources := calcPodResources(&deployment.Spec.Template.Spec)
	newResources := podResources.Containers.MulInt32(*replicas - maxUnavailable).Add(podResources.MaxResources.MulInt32(maxNonReadyPodCount))

	resourceUsage := ResourceUsage{
		Resources: newResources,
		Details: Details{
			Version:     deployment.APIVersion,
			Kind:        deployment.Kind,
			Name:        deployment.Name,
			Replicas:    *replicas,
			Strategy:    string(strategy.Type),
			MaxReplicas: *replicas + maxSurge,
		},
	}

	return &resourceUsage, nil
}
