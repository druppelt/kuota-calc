package calc

import (
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestDeployment(t *testing.T) {
	var tests = []struct {
		name        string
		deployment  string
		cpuMin      resource.Quantity
		cpuMax      resource.Quantity
		memoryMin   resource.Quantity
		memoryMax   resource.Quantity
		replicas    int32
		maxReplicas int32
		strategy    appsv1.DeploymentStrategyType
	}{
		{
			name:        "normal deployment",
			deployment:  normalDeployment,
			cpuMin:      resource.MustParse("2750m"),
			cpuMax:      resource.MustParse("5500m"),
			memoryMin:   resource.MustParse("22Gi"),
			memoryMax:   resource.MustParse("44Gi"),
			replicas:    10,
			maxReplicas: 11,
			strategy:    appsv1.RollingUpdateDeploymentStrategyType,
		},
		{
			name:        "deployment without strategy",
			deployment:  deploymentWithoutStrategy,
			cpuMin:      resource.MustParse("2750m"),
			cpuMax:      resource.MustParse("11"),
			memoryMin:   resource.MustParse("22Gi"),
			memoryMax:   resource.MustParse("44Gi"),
			replicas:    10,
			maxReplicas: 11,
			strategy:    appsv1.RollingUpdateDeploymentStrategyType,
		},
		{
			name:        "deployment with absolute unavailable/surge values",
			deployment:  deploymentWithAbsoluteValues,
			cpuMin:      resource.MustParse("3"),
			cpuMax:      resource.MustParse("12"),
			memoryMin:   resource.MustParse("24Gi"),
			memoryMax:   resource.MustParse("48Gi"),
			replicas:    10,
			maxReplicas: 12,
			strategy:    appsv1.RollingUpdateDeploymentStrategyType,
		},
		{
			name:        "zero replica deployment",
			deployment:  zeroReplicaDeployment,
			cpuMin:      resource.MustParse("0"),
			cpuMax:      resource.MustParse("0"),
			memoryMin:   resource.MustParse("0"),
			memoryMax:   resource.MustParse("0"),
			replicas:    0,
			maxReplicas: 0,
			strategy:    appsv1.RollingUpdateDeploymentStrategyType,
		},
		{
			name:        "recreate deployment",
			deployment:  recrateDeployment,
			cpuMin:      resource.MustParse("2500m"),
			cpuMax:      resource.MustParse("10"),
			memoryMin:   resource.MustParse("20Gi"),
			memoryMax:   resource.MustParse("40Gi"),
			replicas:    10,
			maxReplicas: 10,
			strategy:    appsv1.RecreateDeploymentStrategyType,
		},
		{
			name:        "deployment without max unavailable/surge values",
			deployment:  deploymentWithoutValues,
			cpuMin:      resource.MustParse("2750m"),
			cpuMax:      resource.MustParse("11"),
			memoryMin:   resource.MustParse("22Gi"),
			memoryMax:   resource.MustParse("44Gi"),
			replicas:    10,
			maxReplicas: 11,
			strategy:    appsv1.RollingUpdateDeploymentStrategyType,
		},
		{
			name:       "deployment with init container(s)",
			deployment: initContainerDeployment,
			// TODO the expectation shouldn't be for the init container resources to be considered for all containers,
			//  just the amount that can be in non-ready state simultaneously
			cpuMin:      resource.MustParse("1200m"),
			cpuMax:      resource.MustParse("4400m"),
			memoryMin:   resource.MustParse("8592Mi"),
			memoryMax:   resource.MustParse("17184Mi"),
			replicas:    3,
			maxReplicas: 4,
			strategy:    appsv1.RollingUpdateDeploymentStrategyType,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := require.New(t)

			usage, err := ResourceQuotaFromYaml([]byte(test.deployment))
			r.NoError(err)
			r.NotEmpty(usage)

			AssertEqualQuantities(r, test.cpuMin, *usage.CpuMin, "cpu request value")
			AssertEqualQuantities(r, test.cpuMax, *usage.CpuMax, "cpu limit value")
			AssertEqualQuantities(r, test.memoryMin, *usage.MemoryMin, "memory request value")
			AssertEqualQuantities(r, test.memoryMax, *usage.MemoryMax, "memory limit value")
			r.Equal(test.replicas, usage.Details.Replicas, "replicas")
			r.Equal(string(test.strategy), usage.Details.Strategy, "strategy")
			r.Equal(test.maxReplicas, usage.Details.MaxReplicas, "maxReplicas")
		})
	}
}
