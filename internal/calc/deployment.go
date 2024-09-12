package calc

import (
	"fmt"
	"math"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// calculates the cpu/memory resources a single deployment needs. Replicas and the deployment
// strategy are taken into account.
func deployment(deployment appsv1.Deployment) (*ResourceUsage, error) {
	var (
		resourceOverhead float64 // max overhead compute resources (percent)
		podOverhead      int32   // max overhead pods during deployment
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
		// no overhead on recreate
		resourceOverhead = 1
		podOverhead = 0
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
		maxUnavailable, err := intstr.GetScaledValueFromIntOrPercent(&maxUnavailableValue, int(*replicas), false)
		if err != nil {
			return nil, err
		}

		// docs say, absolute number is calculated by rounding up.
		maxSurge, err := intstr.GetScaledValueFromIntOrPercent(&maxSurgeValue, int(*replicas), true)
		if err != nil {
			return nil, err
		}

		// podOverhead is the number of pods which can run more during a deployment
		podOverheadInt := maxSurge - maxUnavailable
		if podOverheadInt > math.MaxInt32 || podOverheadInt < math.MinInt32 {
			return nil, fmt.Errorf("deployment: %s maxSurge - maxUnavailable (%d-%d) was out of bounds for int32", deployment.Name, maxSurge, maxUnavailable)
		}
		podOverhead = int32(podOverheadInt) //nolint:gosec,wsl // gosec doesn't understand that the int conversion is already guarded, wsl wants to group the assignment with the next block

		resourceOverhead = (float64(podOverhead) / float64(*replicas)) + 1
	default:
		return nil, fmt.Errorf("deployment: %s deployment strategy %q is unknown", deployment.Name, strategy.Type)
	}

	podResources := podResources(&deployment.Spec.Template.Spec)
	newResources := podResources.Mul(float64(*replicas)).Mul(resourceOverhead)

	resourceUsage := ResourceUsage{
		Resources: newResources,
		Details: Details{
			Version:     deployment.APIVersion,
			Kind:        deployment.Kind,
			Name:        deployment.Name,
			Replicas:    *replicas,
			Strategy:    string(strategy.Type),
			MaxReplicas: *replicas + podOverhead,
		},
	}

	return &resourceUsage, nil
}
