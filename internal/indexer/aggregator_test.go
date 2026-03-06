//
// Copyright (C) 2026 Tim Sleptsov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package indexer

import (
	"testing"
	"time"
)

func TestAggregateParentHashes(t *testing.T) {
	agg := NewAggregator()
	parsed := ParsedProfile{
		Info: StackFileInfo{},
		Samples: []Sample{{
			Stack:   []string{"a", "b", "c"},
			Samples: 2,
		}},
	}
	task := Task{Service: "svc", Timestamp: time.Unix(100, 0)}

	block := agg.Aggregate(parsed, task)
	if len(block.Stacks) != 3 {
		t.Fatalf("stacks len = %d", len(block.Stacks))
	}

	h1 := hashFrame(0, "a")
	h2 := hashFrame(h1, "b")
	h3 := hashFrame(h2, "c")

	byHash := make(map[int64]dbStackRecord)
	for _, rec := range block.Stacks {
		byHash[rec.CallStackHash] = dbStackRecord{parent: rec.CallStackParent, name: rec.CallStackName, samples: rec.NumSamples}
	}

	assertRecord(t, byHash, h1, 0, "a", 2)
	assertRecord(t, byHash, h2, h1, "b", 2)
	assertRecord(t, byHash, h3, h2, "c", 2)
}

func TestAggregateWeightsSum(t *testing.T) {
	agg := NewAggregator()
	parsed := ParsedProfile{
		Samples: []Sample{
			{Stack: []string{"a", "b"}, Samples: 3},
			{Stack: []string{"a", "b"}, Samples: 4},
		},
	}
	task := Task{Service: "svc", Timestamp: time.Unix(100, 0)}

	block := agg.Aggregate(parsed, task)
	if len(block.Stacks) != 2 {
		t.Fatalf("stacks len = %d", len(block.Stacks))
	}

	h1 := hashFrame(0, "a")
	h2 := hashFrame(h1, "b")

	byHash := make(map[int64]int64)
	for _, rec := range block.Stacks {
		byHash[rec.CallStackHash] = rec.NumSamples
	}
	if byHash[h1] != 7 || byHash[h2] != 7 {
		t.Fatalf("expected weights 7/7, got %d/%d", byHash[h1], byHash[h2])
	}
}

func TestAggregateKernelSwapperFiltered(t *testing.T) {
	agg := NewAggregator()
	parsed := ParsedProfile{
		Samples: []Sample{{
			Stack:   []string{"swapper/0", "rest"},
			Samples: 10,
		}},
	}
	task := Task{Service: "svc", Timestamp: time.Unix(100, 0)}

	block := agg.Aggregate(parsed, task)
	if len(block.Stacks) != 0 {
		t.Fatalf("expected no stacks, got %d", len(block.Stacks))
	}
}

func TestAggregateMetricZeroPreserved(t *testing.T) {
	agg := NewAggregator()
	parsed := ParsedProfile{}
	parsed.Info.Metrics.CPUAvg = 0
	parsed.Info.Metrics.MemoryAvg = 2.5
	parsed.Info.Metadata.CloudInfo.InstanceType = "c5"
	parsed.Info.Metadata.Hostname = "host"
	task := Task{Service: "svc", Timestamp: time.Unix(100, 0)}

	block := agg.Aggregate(parsed, task)
	if block.Metrics == nil {
		t.Fatalf("expected metrics")
	}
	if block.Metrics.CPUAverageUsedPercent != 0 {
		t.Fatalf("cpu = %v", block.Metrics.CPUAverageUsedPercent)
	}
	if block.Metrics.MemoryAverageUsedPercent != 2.5 {
		t.Fatalf("mem = %v", block.Metrics.MemoryAverageUsedPercent)
	}
}

func TestHashFrameNonNegative(t *testing.T) {
	if h := hashFrame(0, "frame"); h < 0 {
		t.Fatalf("hash is negative: %d", h)
	}
}

type dbStackRecord struct {
	parent  int64
	name    string
	samples int64
}

func assertRecord(t *testing.T, records map[int64]dbStackRecord, hash int64, parent int64, name string, samples int64) {
	rec, ok := records[hash]
	if !ok {
		t.Fatalf("missing record %d", hash)
	}
	if rec.parent != parent || rec.name != name || rec.samples != samples {
		t.Fatalf("record mismatch: parent=%d name=%q samples=%d", rec.parent, rec.name, rec.samples)
	}
}

func BenchmarkHashFrame(b *testing.B) {
	b.ReportAllocs()
	parent := int64(0)
	for i := 0; i < b.N; i++ {
		parent = hashFrame(parent, "frame")
	}
	_ = parent
}
