// Package calc provides function to calculate resource quotas for different k8s resources.
package calc

import (
	"errors"
	"fmt"

	openshiftAppsV1 "github.com/openshift/api/apps/v1"
	openshiftScheme "github.com/openshift/client-go/apps/clientset/versioned/scheme"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	batchV1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	// ErrResourceNotSupported is returned if a k8s resource is not supported by kuota-calc.
	ErrResourceNotSupported = errors.New("resource not supported")
)

// CalculationError is an error implementation that includes a k8s Kind/Version.
type CalculationError struct {
	Version string
	Kind    string
	err     error
}

func (cErr CalculationError) Error() string {
	return fmt.Sprintf(
		"calculating %s/%s resource usage: %s",
		cErr.Version,
		cErr.Kind,
		cErr.err,
	)
}

// Unwrap implements the errors.Unwrap interface.
func (cErr CalculationError) Unwrap() error {
	return cErr.err
}

// ResourceUsage summarizes the usage of compute resources for a k8s resource.
type ResourceUsage struct {
	Resources Resources
	Details   Details
}

// Details contains a few details of a k8s resource, which are needed to generate a detailed resource
// usage report.
type Details struct {
	Version     string
	Kind        string
	Name        string
	Strategy    string
	Replicas    int32
	MaxReplicas int32
}

// Resources contains the limits and requests for cpu and memory that are typically used in kubernetes and openshift.
// Can be used to apply arithmetic operations equally on all quantities.
type Resources struct {
	CPUMin    resource.Quantity
	CPUMax    resource.Quantity
	MemoryMin resource.Quantity
	MemoryMax resource.Quantity
}

// PodResources contain the sum of the resources required by the initContainer, the normal containers
// and the maximum the pod can require at any time for each resource quantity.
// In other words, max(Containers.MinCPU, InitContainers.MinCPU), max(Containers.MaxCPU, InitContainers.MaxCPU), etc.
type PodResources struct {
	Containers     Resources
	InitContainers Resources
	MaxResources   Resources
}

// ConvertToResources converts a kubernetes/openshift ResourceRequirements struct to a Resources struct
func ConvertToResources(req *v1.ResourceRequirements) Resources {
	return Resources{
		CPUMin:    *req.Requests.Cpu(),
		CPUMax:    *req.Limits.Cpu(),
		MemoryMin: *req.Requests.Memory(),
		MemoryMax: *req.Limits.Memory(),
	}
}

// Add adds the provided y resources to the current value.
func (r Resources) Add(y Resources) Resources {
	r.CPUMin.Add(y.CPUMin)
	r.CPUMax.Add(y.CPUMax)
	r.MemoryMin.Add(y.MemoryMin)
	r.MemoryMax.Add(y.MemoryMax)

	return r
}

// MulInt32 multiplies all resource values by the given multiplier.
func (r Resources) MulInt32(y int32) Resources {
	return r.Mul(float64(y))
}

// Mul multiplies all resource values by the given multiplier.
func (r Resources) Mul(y float64) Resources {
	// TODO check if overflow issues due to milli instead of value are to be expected
	r.CPUMin.SetMilli(int64(float64(r.CPUMin.MilliValue()) * y))
	r.CPUMax.SetMilli(int64(float64(r.CPUMax.MilliValue()) * y))
	r.MemoryMin.SetMilli(int64(float64(r.MemoryMin.MilliValue()) * y))
	r.MemoryMax.SetMilli(int64(float64(r.MemoryMax.MilliValue()) * y))

	return r
}

func calcPodResources(podSpec *v1.PodSpec) (r *PodResources) {
	r = new(PodResources)

	for i := range podSpec.Containers {
		container := podSpec.Containers[i]

		r.Containers.CPUMin.Add(*container.Resources.Requests.Cpu())
		r.Containers.CPUMax.Add(*container.Resources.Limits.Cpu())
		r.Containers.MemoryMin.Add(*container.Resources.Requests.Memory())
		r.Containers.MemoryMax.Add(*container.Resources.Limits.Memory())
	}

	for i := range podSpec.InitContainers {
		container := podSpec.InitContainers[i]

		r.InitContainers.CPUMin.Add(*container.Resources.Requests.Cpu())
		r.InitContainers.CPUMax.Add(*container.Resources.Limits.Cpu())
		r.InitContainers.MemoryMin.Add(*container.Resources.Requests.Memory())
		r.InitContainers.MemoryMax.Add(*container.Resources.Limits.Memory())
	}

	r.MaxResources.CPUMin = maxQuantity(r.Containers.CPUMin, r.InitContainers.CPUMin)
	r.MaxResources.CPUMax = maxQuantity(r.Containers.CPUMax, r.InitContainers.CPUMax)
	r.MaxResources.MemoryMin = maxQuantity(r.Containers.MemoryMin, r.InitContainers.MemoryMin)
	r.MaxResources.MemoryMax = maxQuantity(r.Containers.MemoryMax, r.InitContainers.MemoryMax)

	return
}

func maxQuantity(q1, q2 resource.Quantity) resource.Quantity {
	if q1.MilliValue() > q2.MilliValue() {
		return q1
	}

	return q2
}

// ResourceQuotaFromYaml decodes a single yaml document into a k8s object. Then performs a type assertion
// on the object and calculates the resource needs of it.
// Currently supported:
// * apps.openshift.io/v1 - DeploymentConfig
// * apps/v1 - Deployment
// * apps/v1 - StatefulSet
// * apps/v1 - DaemonSet
// * batch/v1 - CronJob
// * batch/v1 - Job
// * v1 - Pod
func ResourceQuotaFromYaml(yamlData []byte) (*ResourceUsage, error) {
	var version string

	var kind string

	combinedScheme := runtime.NewScheme()
	_ = scheme.AddToScheme(combinedScheme)
	_ = openshiftScheme.AddToScheme(combinedScheme)
	codecs := serializer.NewCodecFactory(combinedScheme)
	decoder := codecs.UniversalDeserializer()

	object, gvk, err := decoder.Decode(yamlData, nil, nil)

	if err != nil {
		// when the kind is not found, I just warn and skip
		if runtime.IsNotRegisteredError(err) {
			log.Warn().Msg(err.Error())

			unknown := runtime.Unknown{Raw: yamlData}

			if _, gvk1, err := decoder.Decode(yamlData, nil, &unknown); err == nil {
				kind = gvk1.Kind
				version = gvk1.Version
			}
		} else {
			return nil, fmt.Errorf("decoding yaml data: %w", err)
		}
	} else {
		kind = gvk.Kind
		version = gvk.Version
	}

	switch obj := object.(type) {
	case *openshiftAppsV1.DeploymentConfig:
		usage, err := deploymentConfig(*obj)
		if err != nil {
			return nil, CalculationError{
				Version: gvk.Version,
				Kind:    gvk.Kind,
				err:     err,
			}
		}

		return usage, nil
	case *appsv1.Deployment:
		usage, err := deployment(*obj)
		if err != nil {
			return nil, CalculationError{
				Version: gvk.Version,
				Kind:    gvk.Kind,
				err:     err,
			}
		}

		return usage, nil
	case *appsv1.StatefulSet:
		usage, err := statefulSet(*obj)
		if err != nil {
			return nil, CalculationError{
				Version: gvk.Version,
				Kind:    gvk.Kind,
				err:     err,
			}
		}

		return usage, nil
	case *appsv1.DaemonSet:
		return daemonSet(*obj), nil
	case *batchV1.Job:
		return job(*obj), nil
	case *batchV1.CronJob:
		return cronjob(*obj), nil
	case *v1.Pod:
		return pod(*obj), nil
	default:
		return nil, CalculationError{
			Version: version,
			Kind:    kind,
			err:     ErrResourceNotSupported,
		}
	}
}
