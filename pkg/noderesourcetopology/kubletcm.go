package noderesourcetopology

import (
	v1 "k8s.io/api/core/v1"
	v1qos "k8s.io/kubernetes/pkg/apis/core/v1/helper/qos"
	"k8s.io/kubernetes/pkg/kubelet/cm/topologymanager/bitmask"
)

// TopologyHint is a struct containing the NUMANodeAffinity for a Container
type TopologyHint struct {
	NUMANodeAffinity bitmask.BitMask
	// Preferred is set to true when the NUMANodeAffinity encodes a preferred
	// allocation for the Container. It is set to false otherwise.
	Preferred bool
}

//func GetCpuTopologyHints(pod *v1.Pod, zones topologyv1alpha1.ZoneList, nodeInfo *framework.NodeInfo, container *v1.Container) map[string][]TopologyHint {
//	request := guaranteedCPUs(pod, container)
//	if request == 0 {
//		return nil
//	}
//	//nodes := createNUMANodeList(zones)
//	return nil
//}

//func guaranteedCPUs(pod *v1.Pod, container *v1.Container) int {
//	if v1qos.GetPodQOS(pod) != v1.PodQOSGuaranteed {
//		return 0
//	}
//	cpuQuantity := container.Resources.Requests[v1.ResourceCPU]
//	if cpuQuantity.Value()*1000 != cpuQuantity.MilliValue() {
//		return 0
//	}
//	// Safe downcast to do for all systems with < 2.1 billion CPUs.
//	// Per the language spec, `int` is guaranteed to be at least 32 bits wide.
//	// https://golang.org/ref/spec#Numeric_types
//	return int(cpuQuantity.Value())
//}

//func guaranteedCPUs(nodeInfo *framework.NodeInfo) int64 {
//	//resources := util.GetPodEffectiveRequest(pod)
//	request := nodeInfo.Requested.MilliCPU
//	return request
//}

//func podGuaranteedCPUs(pod *v1.Pod) int {
//	// The maximum of requested CPUs by init containers.
//	requestedByInitContainers := 0
//	for _, container := range pod.Spec.InitContainers {
//		if _, ok := container.Resources.Requests[v1.ResourceCPU]; !ok {
//			continue
//		}
//		requestedCPU := guaranteedCPUs(pod, &container)
//		if requestedCPU > requestedByInitContainers {
//			requestedByInitContainers = requestedCPU
//		}
//	}
//	// The sum of requested CPUs by app containers.
//	requestedByAppContainers := 0
//	for _, container := range pod.Spec.Containers {
//		if _, ok := container.Resources.Requests[v1.ResourceCPU]; !ok {
//			continue
//		}
//		requestedByAppContainers += guaranteedCPUs(pod, &container)
//	}
//
//	if requestedByInitContainers > requestedByAppContainers {
//		return requestedByInitContainers
//	}
//	return requestedByAppContainers
//}
//
func guaranteedCPUs(pod *v1.Pod, container *v1.Container) int {
	if v1qos.GetPodQOS(pod) != v1.PodQOSGuaranteed {
		return 0
	}
	cpuQuantity := container.Resources.Requests[v1.ResourceCPU]
	if cpuQuantity.Value()*1000 != cpuQuantity.MilliValue() {
		return 0
	}
	// Safe downcast to do for all systems with < 2.1 billion CPUs.
	// Per the language spec, `int` is guaranteed to be at least 32 bits wide.
	// https://golang.org/ref/spec#Numeric_types
	return int(cpuQuantity.Value())
}

func guaranteedMemory(pod *v1.Pod, container *v1.Container) int {
	if v1qos.GetPodQOS(pod) != v1.PodQOSGuaranteed {
		return 0
	}
	memoryQuantity := container.Resources.Requests[v1.ResourceMemory]
	if memoryQuantity.Value()*1000 != memoryQuantity.MilliValue() {
		return 0
	}
	// Safe downcast to do for all systems with < 2.1 billion CPUs.
	// Per the language spec, `int` is guaranteed to be at least 32 bits wide.
	// https://golang.org/ref/spec#Numeric_types
	return int(memoryQuantity.Value())
}

func podGuaranteedCPUs(pod *v1.Pod) int {
	// The maximum of requested CPUs by init containers.
	requestedByInitContainers := 0
	for _, container := range pod.Spec.InitContainers {
		if _, ok := container.Resources.Requests[v1.ResourceCPU]; !ok {
			continue
		}
		requestedCPU := guaranteedCPUs(pod, &container)
		if requestedCPU > requestedByInitContainers {
			requestedByInitContainers = requestedCPU
		}
	}
	// The sum of requested CPUs by app containers.
	requestedByAppContainers := 0
	for _, container := range pod.Spec.Containers {
		if _, ok := container.Resources.Requests[v1.ResourceCPU]; !ok {
			continue
		}
		requestedByAppContainers += guaranteedCPUs(pod, &container)
	}

	if requestedByInitContainers > requestedByAppContainers {
		return requestedByInitContainers
	}
	return requestedByAppContainers
}

func podGuaranteedMemory(pod *v1.Pod) int {
	// The maximum of requested CPUs by init containers.
	requestedByInitContainers := 0
	for _, container := range pod.Spec.InitContainers {
		if _, ok := container.Resources.Requests[v1.ResourceMemory]; !ok {
			continue
		}
		requestedCPU := guaranteedMemory(pod, &container)
		if requestedCPU > requestedByInitContainers {
			requestedByInitContainers = requestedCPU
		}
	}
	// The sum of requested CPUs by app containers.
	requestedByAppContainers := 0
	for _, container := range pod.Spec.Containers {
		if _, ok := container.Resources.Requests[v1.ResourceMemory]; !ok {
			continue
		}
		requestedByAppContainers += guaranteedMemory(pod, &container)
	}

	if requestedByInitContainers > requestedByAppContainers {
		return requestedByInitContainers
	}
	return requestedByAppContainers
}
