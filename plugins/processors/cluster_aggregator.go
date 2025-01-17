// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processors

import (
	"github.com/wavefronthq/wavefront-kubernetes-collector/internal/metrics"
)

type ClusterAggregator struct {
	MetricsToAggregate []string
}

func (aggregator *ClusterAggregator) Name() string {
	return "cluster_aggregator"
}

func (aggregator *ClusterAggregator) Process(batch *metrics.DataBatch) (*metrics.DataBatch, error) {
	clusterKey := metrics.ClusterKey()
	cluster := clusterMetricSet()
	for _, metricSet := range batch.MetricSets {
		if metricSetType, found := metricSet.Labels[metrics.LabelMetricSetType.Key]; found &&
			metricSetType == metrics.MetricSetTypeNamespace {
			if err := aggregate(metricSet, cluster, aggregator.MetricsToAggregate); err != nil {
				return nil, err
			}
		}
	}

	batch.MetricSets[clusterKey] = cluster
	return batch, nil
}

func clusterMetricSet() *metrics.MetricSet {
	return &metrics.MetricSet{
		MetricValues: make(map[string]metrics.MetricValue),
		Labels: map[string]string{
			metrics.LabelMetricSetType.Key: metrics.MetricSetTypeCluster,
		},
	}
}
