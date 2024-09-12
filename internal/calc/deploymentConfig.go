package calc

import (
	"fmt"
	"math"

	openshiftAppsV1 "github.com/openshift/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// calculates the cpu/memory resources a single deployment needs. Replicas and the deployment
// strategy are taken into account.
func deploymentConfig(deploymentConfig openshiftAppsV1.DeploymentConfig) (*ResourceUsage, error) {
	var (
		resourceOverhead float64 // max overhead compute resources (percent)
		podOverhead      int32   // max overhead pods during deploymentConfig
	)

	replicas := deploymentConfig.Spec.Replicas
	strategy := deploymentConfig.Spec.Strategy

	if replicas == 0 {
		return &ResourceUsage{
			resources: *new(Resources),
			Details: Details{
				Version:     deploymentConfig.APIVersion,
				Kind:        deploymentConfig.Kind,
				Name:        deploymentConfig.Name,
				Replicas:    replicas,
				MaxReplicas: replicas,
				Strategy:    string(strategy.Type),
			},
		}, nil
	}
	// TODO lookup default values, these are copied from kubernetes Deployment
	switch strategy.Type {
	case openshiftAppsV1.DeploymentStrategyTypeRecreate:
		// no overhead on recreate
		resourceOverhead = 1
		podOverhead = 0
	case "":
		// Rolling is the default and can be an empty string. If so, set the defaults
		// (https://pkg.go.dev/k8s.io/api/apps/v1?tab=doc#RollingUpdateDeployment) and continue calculation.
		defaults := intstr.FromString("25%")
		strategy = openshiftAppsV1.DeploymentStrategy{
			Type: openshiftAppsV1.DeploymentStrategyTypeRolling,
			RollingParams: &openshiftAppsV1.RollingDeploymentStrategyParams{
				MaxUnavailable: &defaults,
				MaxSurge:       &defaults,
			},
		}

		fallthrough
	case openshiftAppsV1.DeploymentStrategyTypeRolling:
		// Documentation: https://pkg.go.dev/k8s.io/api/apps/v1?tab=doc#RollingUpdateDeployment
		// all default values are set as stated in the docs
		var (
			maxUnavailableValue intstr.IntOrString
			maxSurgeValue       intstr.IntOrString
		)

		// can be nil, if so apply default value
		if strategy.RollingParams == nil {
			maxUnavailableValue = intstr.FromString("25%")
			maxSurgeValue = intstr.FromString("25%")
		} else {
			maxUnavailableValue = *strategy.RollingParams.MaxUnavailable
			maxSurgeValue = *strategy.RollingParams.MaxSurge
		}

		// docs say, that the absolute number is calculated by rounding down.
		maxUnavailable, err := intstr.GetScaledValueFromIntOrPercent(&maxUnavailableValue, int(replicas), false)
		if err != nil {
			return nil, err
		}

		// docs say, absolute number is calculated by rounding up.
		maxSurge, err := intstr.GetScaledValueFromIntOrPercent(&maxSurgeValue, int(replicas), true)
		if err != nil {
			return nil, err
		}

		// podOverhead is the number of pods which can run more during a deployment
		podOverheadInt := maxSurge - maxUnavailable
		if podOverheadInt > math.MaxInt32 || podOverheadInt < math.MinInt32 {
			return nil, fmt.Errorf("deploymentConfig: %s maxSurge - maxUnavailable (%d-%d) was out of bounds for int32", deploymentConfig.Name, maxSurge, maxUnavailable)
		}
		podOverhead = int32(podOverheadInt) //nolint:gosec,wsl // gosec doesn't understand that the int conversion is already guarded, wsl wants to group the assignment with the next block

		resourceOverhead = (float64(podOverhead) / float64(replicas)) + 1
	default:
		return nil, fmt.Errorf("deploymentConfig: %s deploymentConfig strategy %q is unknown", deploymentConfig.Name, strategy.Type)
	}

	podResources := podResources(&deploymentConfig.Spec.Template.Spec)
	strategyResources := ConvertToResources(&deploymentConfig.Spec.Strategy.Resources)
	newResources := (*podResources).Mul(float64(replicas)).Mul(resourceOverhead).Add(strategyResources)

	resourceUsage := ResourceUsage{
		resources: newResources,
		Details: Details{
			Version:     deploymentConfig.APIVersion,
			Kind:        deploymentConfig.Kind,
			Name:        deploymentConfig.Name,
			Replicas:    replicas,
			Strategy:    string(strategy.Type),
			MaxReplicas: replicas + podOverhead,
		},
	}

	return &resourceUsage, nil
}
