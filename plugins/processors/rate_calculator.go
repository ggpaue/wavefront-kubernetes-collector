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

	"github.com/golang/glog"
)

type RateCalculator struct {
	rateMetricsMapping map[string]metrics.Metric
	previousMetricSets map[string]*metrics.MetricSet
}

func (this *RateCalculator) Name() string {
	return "rate calculator"
}

func (this *RateCalculator) Process(batch *metrics.DataBatch) (*metrics.DataBatch, error) {
	for key, newMs := range batch.MetricSets {
		oldMs, found := this.previousMetricSets[key]
		if !found {
			glog.V(4).Infof("Skipping rate calculation for '%s' - no previous batch found", key)
			this.previousMetricSets[key] = newMs
			continue
		}

		if !newMs.ScrapeTime.After(oldMs.ScrapeTime) {
			// New must be strictly after old.
			glog.V(4).Infof("Skipping rate calculations for %'s' - new batch (%s) was not scraped strictly after old batch (%s)", key, newMs.ScrapeTime, oldMs.ScrapeTime)
			continue
		}
		if !newMs.CollectionStartTime.Equal(oldMs.CollectionStartTime) {
			glog.V(4).Infof("Skipping rates for %s - different collection start time (restart) new:%v  old:%v", key, newMs.CollectionStartTime, oldMs.CollectionStartTime)
			this.previousMetricSets[key] = newMs
			continue
		}

		var metricValNew, metricValOld metrics.MetricValue
		var foundNew, foundOld bool

		for metricName, targetMetric := range this.rateMetricsMapping {
			if metricName == metrics.MetricDiskIORead.MetricDescriptor.Name || metricName == metrics.MetricDiskIOWrite.MetricDescriptor.Name {
				for _, itemNew := range newMs.LabeledMetrics {
					foundNew, foundOld = false, false
					if itemNew.Name == metricName {
						metricValNew, foundNew = itemNew.MetricValue, true
						for _, itemOld := range oldMs.LabeledMetrics {
							// Fix negative value on "disk/io_read_bytes_rate" and "disk/io_write_bytes_rate" when multiple disk devices are available
							if itemOld.Name == metricName && itemOld.Labels[metrics.LabelResourceID.Key] == itemNew.Labels[metrics.LabelResourceID.Key] {
								metricValOld, foundOld = itemOld.MetricValue, true
								break
							}
						}
					}

					if foundNew && foundOld {
						if targetMetric.MetricDescriptor.ValueType == metrics.ValueFloat {
							newVal := 1e9 * float64(metricValNew.IntValue-metricValOld.IntValue) /
								float64(newMs.ScrapeTime.UnixNano()-oldMs.ScrapeTime.UnixNano())

							newMs.LabeledMetrics = append(newMs.LabeledMetrics, metrics.LabeledMetric{
								Name:   targetMetric.MetricDescriptor.Name,
								Labels: itemNew.Labels,
								MetricValue: metrics.MetricValue{
									ValueType:  metrics.ValueFloat,
									MetricType: metrics.MetricGauge,
									FloatValue: newVal,
								},
							})
						}
					} else if foundNew && !foundOld || !foundNew && foundOld {
						glog.V(4).Infof("Skipping rates for %s in %s: metric not found in one of old (%v) or new (%v)", metricName, key, foundOld, foundNew)
					}
				}
			} else {
				metricValNew, foundNew = newMs.MetricValues[metricName]
				metricValOld, foundOld = oldMs.MetricValues[metricName]

				if foundNew && foundOld && metricName == metrics.MetricCpuUsage.MetricDescriptor.Name {
					// cpu/usage values are in nanoseconds; we want to have it in millicores (that's why constant 1000 is here).
					newVal := 1000 * (metricValNew.IntValue - metricValOld.IntValue) /
						(newMs.ScrapeTime.UnixNano() - oldMs.ScrapeTime.UnixNano())

					newMs.MetricValues[targetMetric.MetricDescriptor.Name] = metrics.MetricValue{
						ValueType:  metrics.ValueInt64,
						MetricType: metrics.MetricGauge,
						IntValue:   newVal,
					}

				} else if foundNew && foundOld && targetMetric.MetricDescriptor.ValueType == metrics.ValueFloat {
					newVal := 1e9 * float64(metricValNew.IntValue-metricValOld.IntValue) /
						float64(newMs.ScrapeTime.UnixNano()-oldMs.ScrapeTime.UnixNano())

					newMs.MetricValues[targetMetric.MetricDescriptor.Name] = metrics.MetricValue{
						ValueType:  metrics.ValueFloat,
						MetricType: metrics.MetricGauge,
						FloatValue: newVal,
					}
				} else if foundNew && !foundOld || !foundNew && foundOld {
					glog.V(4).Infof("Skipping rates for %s in %s: metric not found in one of old (%v) or new (%v)", metricName, key, foundOld, foundNew)
				}
			}
		}
		this.previousMetricSets[key] = newMs
	}
	return batch, nil
}

func NewRateCalculator(rateMetricsMapping map[string]metrics.Metric) *RateCalculator {
	return &RateCalculator{
		rateMetricsMapping: rateMetricsMapping,
		previousMetricSets: make(map[string]*metrics.MetricSet, 0),
	}
}
