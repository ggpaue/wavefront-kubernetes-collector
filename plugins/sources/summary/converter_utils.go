package summary

import (
	"sort"

	"github.com/wavefronthq/wavefront-kubernetes-collector/internal/metrics"
)

const (
	sysSubContainerName = "system.slice/"
)

var (
	excludeTagList = [...]string{"namespace_id", "host_id", "pod_id", "hostname"}
)

func sortedMetricSetKeys(m map[string]*metrics.MetricSet) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func sortedMetricValueKeys(m map[string]metrics.MetricValue) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func sortedLabelKeys(m map[string]string) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func processTags(tags map[string]string) {
	for k, v := range tags {
		// ignore tags with empty values as well so the data point doesn't fail validation
		if excludeTag(k) || len(v) == 0 {
			delete(tags, k)
		}
	}
}

func excludeTag(a string) bool {
	for _, b := range excludeTagList {
		if b == a {
			return true
		}
	}
	return false
}
