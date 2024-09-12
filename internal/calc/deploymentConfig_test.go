package calc

import (
	"testing"

	openshiftAppsV1 "github.com/openshift/api/apps/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestDeploymentConfig(t *testing.T) {
	var tests = []struct {
		name             string
		deploymentConfig string
		cpuMin           resource.Quantity
		cpuMax           resource.Quantity
		memoryMin        resource.Quantity
		memoryMax        resource.Quantity
		replicas         int32
		maxReplicas      int32
		strategy         openshiftAppsV1.DeploymentStrategyType
	}{
		{
			name:             "normal deploymentConfig",
			deploymentConfig: normalDeploymentConfig,
			cpuMin:           resource.MustParse("2750m"),
			cpuMax:           resource.MustParse("5500m"),
			memoryMin:        resource.MustParse("22Gi"),
			memoryMax:        resource.MustParse("44Gi"),
			replicas:         10,
			maxReplicas:      11,
			strategy:         openshiftAppsV1.DeploymentStrategyTypeRolling,
		},
		//TODO add more tests
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := require.New(t)

			usage, err := ResourceQuotaFromYaml([]byte(test.deploymentConfig))
			r.NoError(err)
			r.NotEmpty(usage)

			AssertEqualQuantities(r, test.cpuMin, usage.Resources.CPUMin, "cpu request value")
			AssertEqualQuantities(r, test.cpuMax, usage.Resources.CPUMax, "cpu limit value")
			AssertEqualQuantities(r, test.memoryMin, usage.Resources.MemoryMin, "memory request value")
			AssertEqualQuantities(r, test.memoryMax, usage.Resources.MemoryMax, "memory limit value")
			r.Equal(test.replicas, usage.Details.Replicas, "replicas")
			r.Equal(string(test.strategy), usage.Details.Strategy, "strategy")
			r.Equal(test.maxReplicas, usage.Details.MaxReplicas, "maxReplicas")
		})
	}
}
