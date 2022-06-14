package noderesourcetopology

import (
	topologyv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

func GetContainerCpuTopologyHints(pod *v1.Pod, container *v1.Container, zones topologyv1alpha1.ZoneList) map[string][]TopologyHint {
	request := guaranteedCPUs(pod, container)
	if request == 0 {
		return nil
	}
	cpuHints := generateCPUTopologyHints(zones, request)
	//for _, v := range cpuHints {
	//	t := v.NUMANodeAffinity.String()
	//	t += "-wzh"
	//}

	return map[string][]TopologyHint{
		string(v1.ResourceCPU): cpuHints,
	}
}

func GetContainerTopologyHints(pod *v1.Pod, container *v1.Container, zones topologyv1alpha1.ZoneList) map[string][]TopologyHint {
	request := guaranteedMemory(pod, container)
	if request == 0 {
		return nil
	}
	memoryHints := generatememoryTopologyHints(zones, request)
	for _, v := range memoryHints {
		t := v.NUMANodeAffinity.String()
		t += "-wzh"
	}

	return map[string][]TopologyHint{
		string(v1.ResourceMemory): memoryHints,
	}
}
