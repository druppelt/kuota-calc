package calc

import (
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestStatefulSet(t *testing.T) {
	var tests = []struct {
		name        string
		statefulset string
		cpuMin      resource.Quantity
		cpuMax      resource.Quantity
		memoryMin   resource.Quantity
		memoryMax   resource.Quantity
		replicas    int32
		maxReplicas int32
		strategy    appsv1.StatefulSetUpdateStrategyType
	}{
		{
			name:        "ok",
			statefulset: normalStatefulSet,
			cpuMin:      resource.MustParse("500m"),
			cpuMax:      resource.MustParse("2"),
			memoryMin:   resource.MustParse("4Gi"),
			memoryMax:   resource.MustParse("8Gi"),
			replicas:    2,
			maxReplicas: 2,
			strategy:    appsv1.RollingUpdateStatefulSetStrategyType,
		},
		{
			name:        "no replicas",
			statefulset: noReplicasStatefulSet,
			cpuMin:      resource.MustParse("250m"),
			cpuMax:      resource.MustParse("1"),
			memoryMin:   resource.MustParse("2Gi"),
			memoryMax:   resource.MustParse("4Gi"),
			replicas:    1,
			maxReplicas: 1,
			strategy:    appsv1.RollingUpdateStatefulSetStrategyType,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := require.New(t)

			usage, err := ResourceQuotaFromYaml([]byte(test.statefulset))
			r.NoError(err)
			r.NotEmpty(usage)

			AssertEqualQuantities(r, test.cpuMin, usage.resources.CPUMin, "cpu request value")
			AssertEqualQuantities(r, test.cpuMax, usage.resources.CPUMax, "cpu limit value")
			AssertEqualQuantities(r, test.memoryMin, usage.resources.MemoryMin, "memory request value")
			AssertEqualQuantities(r, test.memoryMax, usage.resources.MemoryMax, "memory limit value")
			r.Equalf(test.replicas, usage.Details.Replicas, "replicas")
			r.Equalf(test.maxReplicas, usage.Details.MaxReplicas, "maxReplicas")
			r.Equalf(string(test.strategy), usage.Details.Strategy, "strategy")
		})
	}
}
