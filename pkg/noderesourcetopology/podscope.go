package noderesourcetopology

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm/topologymanager/bitmask"
)

import (
	topologyv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
	"sort"
)

func AdmitPodbestEffort(pod *v1.Pod, zones topologyv1alpha1.ZoneList) bool {
	admit := calculateAffinitypodbestEffort(pod, zones)
	return admit
}

func calculateAffinitypodbestEffort(pod *v1.Pod, zones topologyv1alpha1.ZoneList) bool {
	providersHints := accumulatepodProvidersHints(pod, zones)
	_, admit := MergepodbestEffort(providersHints, zones)
	return admit
	//return false
}

func accumulatepodProvidersHints(pod *v1.Pod, zones topologyv1alpha1.ZoneList) []map[string][]TopologyHint {
	var providersHints []map[string][]TopologyHint
	providersHints = append(providersHints, GetPodCpuTopologyHints(pod, zones))
	providersHints = append(providersHints, GetPodMemoryTopologyHints(pod, zones))
	return providersHints
}

func GetPodCpuTopologyHints(pod *v1.Pod, zones topologyv1alpha1.ZoneList) map[string][]TopologyHint {
	request := podGuaranteedCPUs(pod)
	if request == 0 {
		return nil
	}
	cpuHints := generateCPUTopologyHints(zones, request)
	for _, v := range cpuHints {
		t := v.NUMANodeAffinity.String()
		t += "-wzh"
	}

	return map[string][]TopologyHint{
		string(v1.ResourceCPU): cpuHints,
	}
}

func generateCPUTopologyHints(zones topologyv1alpha1.ZoneList, request int) []TopologyHint {
	nodes := createNUMANodeListCapacity(zones)
	nodesAvailable := createNUMANodeList(zones)
	minAffinitySize := len(nodes)
	hints := []TopologyHint{}
	bitmask.IterateBitMasks(ToSlice(nodes), func(mask bitmask.BitMask) {
		// First, update minAffinitySize for the current request size.
		cpusInMask := getCPUQuantityInMask(mask, nodes)
		if cpusInMask >= int64(request) && mask.Count() < minAffinitySize {
			minAffinitySize = mask.Count()
		}
		// Finally, check to see if enough available CPUs remain on the current
		// NUMA node combination to satisfy the CPU request.
		numMatching := getCPUQuantityInMask(mask, nodesAvailable)

		// If they don't, then move onto the next combination.
		if numMatching < int64(request) {
			return
		}

		// Otherwise, create a new hint from the numa node bitmask and add it to the
		// list of hints.  We set all hint preferences to 'false' on the first
		// pass through.
		hints = append(hints, TopologyHint{
			NUMANodeAffinity: mask,
			Preferred:        false,
		})
	})

	// Loop back through all hints and update the 'Preferred' field based on
	// counting the number of bits sets in the affinity mask and comparing it
	// to the minAffinitySize. Only those with an equal number of bits set (and
	// with a minimal set of numa nodes) will be considered preferred.
	for i := range hints {
		if hints[i].NUMANodeAffinity.Count() == minAffinitySize {
			hints[i].Preferred = true
		}
	}

	return hints
}

func GetPodMemoryTopologyHints(pod *v1.Pod, zones topologyv1alpha1.ZoneList) map[string][]TopologyHint {
	request := podGuaranteedMemory(pod)
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

func generatememoryTopologyHints(zones topologyv1alpha1.ZoneList, request int) []TopologyHint {
	nodes := createNUMANodeListCapacity(zones)
	nodesAvailable := createNUMANodeList(zones)
	minAffinitySize := len(nodes)
	hints := []TopologyHint{}
	bitmask.IterateBitMasks(ToSlice(nodes), func(mask bitmask.BitMask) {
		// First, update minAffinitySize for the current request size.
		cpusInMask := getmemoryQuantityInMask(mask, nodes)
		if cpusInMask >= int64(request) && mask.Count() < minAffinitySize {
			minAffinitySize = mask.Count()
		}
		// Finally, check to see if enough available CPUs remain on the current
		// NUMA node combination to satisfy the CPU request.
		numMatching := getmemoryQuantityInMask(mask, nodesAvailable)

		// If they don't, then move onto the next combination.
		if numMatching < int64(request) {
			return
		}

		// Otherwise, create a new hint from the numa node bitmask and add it to the
		// list of hints.  We set all hint preferences to 'false' on the first
		// pass through.
		hints = append(hints, TopologyHint{
			NUMANodeAffinity: mask,
			Preferred:        false,
		})
	})

	// Loop back through all hints and update the 'Preferred' field based on
	// counting the number of bits sets in the affinity mask and comparing it
	// to the minAffinitySize. Only those with an equal number of bits set (and
	// with a minimal set of numa nodes) will be considered preferred.
	for i := range hints {
		if hints[i].NUMANodeAffinity.Count() == minAffinitySize {
			hints[i].Preferred = true
		}
	}

	return hints
}

func ToSlice(numanodelist NUMANodeList) []int {
	result := []int{}
	for _, numanode := range numanodelist {
		result = append(result, numanode.NUMAID)
	}
	sort.Ints(result)
	return result
}

func createNUMANodeListCapacity(zones topologyv1alpha1.ZoneList) NUMANodeList {
	nodes := make(NUMANodeList, 0)
	for _, zone := range zones {
		if zone.Type == "Node" {
			var numaID int
			_, err := fmt.Sscanf(zone.Name, "node-%d", &numaID)
			if err != nil {
				klog.ErrorS(nil, "Invalid zone format", "zone", zone.Name)
				continue
			}
			if numaID > 63 || numaID < 0 {
				klog.ErrorS(nil, "Invalid NUMA id range", "numaID", numaID)
				continue
			}
			resources := extractResourcesCapacity(zone)
			klog.V(6).InfoS("extracted NUMA resources", resourceListToLoggable(zone.Name, resources)...)
			nodes = append(nodes, NUMANode{NUMAID: numaID, Resources: resources})
		}
	}
	return nodes
}

func extractResourcesCapacity(zone topologyv1alpha1.Zone) v1.ResourceList {
	res := make(v1.ResourceList)
	for _, resInfo := range zone.Resources {
		res[v1.ResourceName(resInfo.Name)] = resInfo.Capacity
	}
	return res
}

func getCPUQuantityInMask(mask bitmask.BitMask, numaNodes NUMANodeList) int64 {
	cpusInMask := int64(0)
	for _, numanodeid := range mask.GetBits() {
		for _, numaNode := range numaNodes {
			if numaNode.NUMAID == numanodeid {
				numaQuantity, ok := numaNode.Resources[v1.ResourceCPU]
				if ok {
					cpusInMask += numaQuantity.Value()
				}
			}
		}
	}
	return cpusInMask
}

func getmemoryQuantityInMask(mask bitmask.BitMask, numaNodes NUMANodeList) int64 {
	cpusInMask := int64(0)
	for _, numanodeid := range mask.GetBits() {
		for _, numaNode := range numaNodes {
			if numaNode.NUMAID == numanodeid {
				numaQuantity, ok := numaNode.Resources[v1.ResourceMemory]
				if ok {
					cpusInMask += numaQuantity.Value()
				}
			}
		}
	}
	return cpusInMask
}

func mergeFilteredHints(numaNodes []int, filteredHints [][]TopologyHint) TopologyHint {
	// Set the default affinity as an any-numa affinity containing the list
	// of NUMA Nodes available on this machine.
	defaultAffinity, _ := bitmask.NewBitMask(numaNodes...)

	// Set the bestHint to return from this function as {nil false}.
	// This will only be returned if no better hint can be found when
	// merging hints from each hint provider.
	bestHint := TopologyHint{defaultAffinity, false}
	iterateAllProviderTopologyHints(filteredHints, func(permutation []TopologyHint) {
		// Get the NUMANodeAffinity from each hint in the permutation and see if any
		// of them encode unpreferred allocations.
		mergedHint := mergePermutation(numaNodes, permutation)
		// Only consider mergedHints that result in a NUMANodeAffinity > 0 to
		// replace the current bestHint.
		if mergedHint.NUMANodeAffinity.Count() == 0 {
			return
		}

		// If the current bestHint is non-preferred and the new mergedHint is
		// preferred, always choose the preferred hint over the non-preferred one.
		if mergedHint.Preferred && !bestHint.Preferred {
			bestHint = mergedHint
			return
		}

		// If the current bestHint is preferred and the new mergedHint is
		// non-preferred, never update bestHint, regardless of mergedHint's
		// narowness.
		if !mergedHint.Preferred && bestHint.Preferred {
			return
		}

		// If mergedHint and bestHint has the same preference, only consider
		// mergedHints that have a narrower NUMANodeAffinity than the
		// NUMANodeAffinity in the current bestHint.
		if !mergedHint.NUMANodeAffinity.IsNarrowerThan(bestHint.NUMANodeAffinity) {
			return
		}

		// In all other cases, update bestHint to the current mergedHint
		bestHint = mergedHint
	})

	return bestHint
}

func mergePermutation(numaNodes []int, permutation []TopologyHint) TopologyHint {
	// Get the NUMANodeAffinity from each hint in the permutation and see if any
	// of them encode unpreferred allocations.
	preferred := true
	defaultAffinity, _ := bitmask.NewBitMask(numaNodes...)
	var numaAffinities []bitmask.BitMask
	for _, hint := range permutation {
		// Only consider hints that have an actual NUMANodeAffinity set.
		if hint.NUMANodeAffinity == nil {
			numaAffinities = append(numaAffinities, defaultAffinity)
		} else {
			numaAffinities = append(numaAffinities, hint.NUMANodeAffinity)
		}

		if !hint.Preferred {
			preferred = false
		}
	}

	// Merge the affinities using a bitwise-and operation.
	mergedAffinity := bitmask.And(defaultAffinity, numaAffinities...)
	// Build a mergedHint from the merged affinity mask, indicating if an
	// preferred allocation was used to generate the affinity mask or not.
	return TopologyHint{mergedAffinity, preferred}
}

func iterateAllProviderTopologyHints(allProviderHints [][]TopologyHint, callback func([]TopologyHint)) {
	// Internal helper function to accumulate the permutation before calling the callback.
	var iterate func(i int, accum []TopologyHint)
	iterate = func(i int, accum []TopologyHint) {
		// Base case: we have looped through all providers and have a full permutation.
		if i == len(allProviderHints) {
			callback(accum)
			return
		}

		// Loop through all hints for provider 'i', and recurse to build the
		// the permutation of this hint with all hints from providers 'i++'.
		for j := range allProviderHints[i] {
			iterate(i+1, append(accum, allProviderHints[i][j]))
		}
	}
	iterate(0, []TopologyHint{})
}

func MergepodbestEffort(providersHints []map[string][]TopologyHint, zones topologyv1alpha1.ZoneList) (TopologyHint, bool) {
	filteredHints := filterProvidersHints(providersHints)
	nodesAvailable := createNUMANodeList(zones)
	var numaNodes []int
	for _, node := range nodesAvailable {
		numaNodes = append(numaNodes, node.NUMAID)
	}
	hint := mergeFilteredHints(numaNodes, filteredHints)
	//admit := hint.Preferred
	admit := canAdmitPodResult(&hint)
	return hint, admit
}

func canAdmitPodResult(hint *TopologyHint) bool {
	return hint.Preferred
	//return true
}

func filterProvidersHints(providersHints []map[string][]TopologyHint) [][]TopologyHint {
	// Loop through all hint providers and save an accumulated list of the
	// hints returned by each hint provider. If no hints are provided, assume
	// that provider has no preference for topology-aware allocation.
	var allProviderHints [][]TopologyHint
	for _, hints := range providersHints {
		// If hints is nil, insert a single, preferred any-numa hint into allProviderHints.
		if len(hints) == 0 {
			klog.InfoS("Hint Provider has no preference for NUMA affinity with any resource")
			allProviderHints = append(allProviderHints, []TopologyHint{{nil, true}})
			continue
		}

		// Otherwise, accumulate the hints for each resource type into allProviderHints.
		for resource := range hints {
			if hints[resource] == nil {
				klog.InfoS("Hint Provider has no preference for NUMA affinity with resource", "resource", resource)
				allProviderHints = append(allProviderHints, []TopologyHint{{nil, true}})
				continue
			}

			if len(hints[resource]) == 0 {
				klog.InfoS("Hint Provider has no possible NUMA affinities for resource", "resource", resource)
				allProviderHints = append(allProviderHints, []TopologyHint{{nil, false}})
				continue
			}

			allProviderHints = append(allProviderHints, hints[resource])
		}
	}
	return allProviderHints
}
