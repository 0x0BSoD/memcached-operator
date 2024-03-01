package controller

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/api/resource"

	cachev1 "github.com/0x0BSoD/memcached-operator/api/v1"
)

func generateResourceRequirements(
	resources *cachev1.Resources,
	defaultResources cachev1.Resources,
	containerName string) (*v1.ResourceRequirements, error) {
	var err error
	specRequests := cachev1.ResourceDescription{}
	specLimits := cachev1.ResourceDescription{}
	result := v1.ResourceRequirements{}

	if resources != nil {
		specRequests = resources.ResourceRequests
		specLimits = resources.ResourceLimits
	}

	result.Requests, err = fillResourceList(specRequests, defaultResources.ResourceRequests)
	if err != nil {
		return nil, fmt.Errorf("could not fill resource requests: %v", err)
	}

	result.Limits, err = fillResourceList(specLimits, defaultResources.ResourceLimits)
	if err != nil {
		return nil, fmt.Errorf("could not fill resource limits: %v", err)
	}

	// make sure after reflecting default and enforcing min limit values we don't have requests > limits
	matchLimitsWithRequestsIfSmaller(&result, containerName)

	return &result, nil
}

func matchLimitsWithRequestsIfSmaller(resources *v1.ResourceRequirements, containerName string) {
	requests := resources.Requests
	limits := resources.Limits
	requestCPU, cpuRequestsExists := requests[v1.ResourceCPU]
	limitCPU, cpuLimitExists := limits[v1.ResourceCPU]
	if cpuRequestsExists && cpuLimitExists && limitCPU.Cmp(requestCPU) == -1 {
		fmt.Printf("CPU limit of %s for %q container is increased to match CPU requests of %s", limitCPU.String(), containerName, requestCPU.String())
		resources.Limits[v1.ResourceCPU] = requestCPU
	}

	requestMemory, memoryRequestsExists := requests[v1.ResourceMemory]
	limitMemory, memoryLimitExists := limits[v1.ResourceMemory]
	if memoryRequestsExists && memoryLimitExists && limitMemory.Cmp(requestMemory) == -1 {
		fmt.Printf("memory limit of %s for %q container is increased to match memory requests of %s", limitMemory.String(), containerName, requestMemory.String())
		resources.Limits[v1.ResourceMemory] = requestMemory
	}
}

func fillResourceList(spec cachev1.ResourceDescription, defaults cachev1.ResourceDescription) (v1.ResourceList, error) {
	var err error
	requests := v1.ResourceList{}
	emptyResourceExamples := []string{"", "0", "null"}

	if spec.CPU != nil && !slices.Contains(emptyResourceExamples, *spec.CPU) {
		requests[v1.ResourceCPU], err = resource.ParseQuantity(*spec.CPU)
		if err != nil {
			return nil, fmt.Errorf("could not parse CPU quantity: %v", err)
		}
	} else {
		if defaults.CPU != nil && !slices.Contains(emptyResourceExamples, *defaults.CPU) {
			requests[v1.ResourceCPU], err = resource.ParseQuantity(*defaults.CPU)
			if err != nil {
				return nil, fmt.Errorf("could not parse default CPU quantity: %v", err)
			}
		}
	}
	if spec.Memory != nil && !slices.Contains(emptyResourceExamples, *spec.Memory) {
		requests[v1.ResourceMemory], err = resource.ParseQuantity(*spec.Memory)
		if err != nil {
			return nil, fmt.Errorf("could not parse memory quantity: %v", err)
		}
	} else {
		if defaults.Memory != nil && !slices.Contains(emptyResourceExamples, *defaults.Memory) {
			requests[v1.ResourceMemory], err = resource.ParseQuantity(*defaults.Memory)
			if err != nil {
				return nil, fmt.Errorf("could not parse default memory quantity: %v", err)
			}
		}
	}

	return requests, nil
}

func makeDefaultResources() cachev1.Resources {
	cpu := "100m"
	mem := "256Mi"

	defaultRequests := cachev1.ResourceDescription{
		CPU:    &cpu,
		Memory: &mem,
	}
	defaultLimits := cachev1.ResourceDescription{
		CPU:    &cpu,
		Memory: &mem,
	}

	return cachev1.Resources{
		ResourceRequests: defaultRequests,
		ResourceLimits:   defaultLimits,
	}
}
