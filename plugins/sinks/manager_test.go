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

package sinks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/wavefronthq/wavefront-kubernetes-collector/internal/metrics"
	"github.com/wavefronthq/wavefront-kubernetes-collector/internal/util"
)

func TestAllExportsInTime(t *testing.T) {
	timeout := 3 * time.Second

	sink1 := util.NewDummySink("s1", time.Second)
	sink2 := util.NewDummySink("s2", time.Second)
	manager, _ := NewDataSinkManager([]metrics.DataSink{sink1, sink2}, timeout, timeout)

	now := time.Now()
	batch := metrics.DataBatch{
		Timestamp: now,
	}

	manager.ExportData(&batch)
	manager.ExportData(&batch)
	manager.ExportData(&batch)

	elapsed := time.Now().Sub(now)
	if elapsed > 3*timeout+2*time.Second {
		t.Fatalf("3xExportData took too long: %s", elapsed)
	}

	time.Sleep(time.Second)
	assert.Equal(t, 3, sink1.GetExportCount())
	assert.Equal(t, 3, sink2.GetExportCount())
}

func TestOneExportInTime(t *testing.T) {
	timeout := 3 * time.Second

	sink1 := util.NewDummySink("s1", time.Second)
	sink2 := util.NewDummySink("s2", 30*time.Second)
	manager, _ := NewDataSinkManager([]metrics.DataSink{sink1, sink2}, timeout, timeout)

	now := time.Now()
	batch := metrics.DataBatch{
		Timestamp: now,
	}

	manager.ExportData(&batch)
	manager.ExportData(&batch)
	manager.ExportData(&batch)

	elapsed := time.Now().Sub(now)
	if elapsed > 2*timeout+2*time.Second {
		t.Fatalf("3xExportData took too long: %s", elapsed)
	}
	if elapsed < 2*timeout-1*time.Second {
		t.Fatalf("3xExportData took too short: %s", elapsed)
	}

	time.Sleep(time.Second)
	assert.Equal(t, 3, sink1.GetExportCount())
	assert.Equal(t, 1, sink2.GetExportCount())
}

func TestNoExportInTime(t *testing.T) {
	timeout := 3 * time.Second

	sink1 := util.NewDummySink("s1", 30*time.Second)
	sink2 := util.NewDummySink("s2", 30*time.Second)
	manager, _ := NewDataSinkManager([]metrics.DataSink{sink1, sink2}, timeout, timeout)

	now := time.Now()
	batch := metrics.DataBatch{
		Timestamp: now,
	}

	manager.ExportData(&batch)
	manager.ExportData(&batch)
	manager.ExportData(&batch)

	elapsed := time.Now().Sub(now)
	if elapsed > 2*timeout+2*time.Second {
		t.Fatalf("3xExportData took too long: %s", elapsed)
	}
	if elapsed < 2*timeout-1*time.Second {
		t.Fatalf("3xExportData took too short: %s", elapsed)
	}

	time.Sleep(time.Second)
	assert.Equal(t, 1, sink1.GetExportCount())
	assert.Equal(t, 1, sink2.GetExportCount())
}

func TestStop(t *testing.T) {
	timeout := 3 * time.Second

	sink1 := util.NewDummySink("s1", 30*time.Second)
	sink2 := util.NewDummySink("s2", 30*time.Second)
	manager, _ := NewDataSinkManager([]metrics.DataSink{sink1, sink2}, timeout, timeout)

	now := time.Now()
	manager.Stop()
	elapsed := time.Now().Sub(now)
	if elapsed > time.Second {
		t.Fatalf("stop too long: %s", elapsed)
	}
	time.Sleep(time.Second)

	assert.Equal(t, true, sink1.IsStopped())
	assert.Equal(t, true, sink2.IsStopped())
}
