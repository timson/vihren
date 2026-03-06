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
	"encoding/binary"
	"io"
	"strings"
	"time"

	"vihren/internal/db"

	"github.com/OneOfOne/xxhash"
)

type ContainerFrameWeights map[string]map[int64]int64

type FrameNode struct {
	Name       string
	Hash       int64
	ParentHash int64
}

type Aggregator struct {
}

func NewAggregator() *Aggregator {
	return &Aggregator{}
}

func (a *Aggregator) Aggregate(profile ParsedProfile, task Task) db.ProfileBlock {
	weights := make(ContainerFrameWeights)
	nodes := make(map[int64]FrameNode)

	for _, sample := range profile.Samples {
		if sample.Samples == 0 || len(sample.Stack) == 0 || isKernelSwapper(sample.Stack) {
			continue
		}
		addStack(weights, nodes, sample.RawContainer, sample.Stack, sample.Samples)
	}

	stacks := buildStackRecords(weights, nodes, task.Service, profile.Info.Metadata.CloudInfo.InstanceType, profile.Info.Metadata.Hostname, task.Timestamp)
	metric := buildMetricRecord(task.Service, profile.Info.Metadata.CloudInfo.InstanceType, profile.Info.Metadata.Hostname, task.Timestamp, profile.Info.Metrics.CPUAvg, profile.Info.Metrics.MemoryAvg)

	return db.ProfileBlock{
		Stacks:    stacks,
		Metrics:   metric,
		Timestamp: task.Timestamp,
	}
}

func isKernelSwapper(stack []string) bool {
	return len(stack) > 0 && strings.HasPrefix(stack[0], "swapper")
}

func addStack(weights ContainerFrameWeights, nodes map[int64]FrameNode, rawContainer string, stack []string, samples int64) {
	if weights[rawContainer] == nil {
		weights[rawContainer] = make(map[int64]int64)
	}

	var parentHash int64
	hasher := xxhash.New64()
	for _, frameName := range stack {
		hash := hashFrameWithHasher(hasher, parentHash, frameName)

		if _, ok := nodes[hash]; !ok {
			nodes[hash] = FrameNode{
				Name:       frameName,
				Hash:       hash,
				ParentHash: parentHash,
			}
		}

		weights[rawContainer][hash] += samples
		parentHash = hash
	}
}

func hashFrame(parentHash int64, frameName string) int64 {
	return hashFrameWithHasher(xxhash.New64(), parentHash, frameName)
}

func hashFrameWithHasher(h *xxhash.XXHash64, parentHash int64, frameName string) int64 {
	h.Reset()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(parentHash))
	_, _ = h.Write(buf[:])
	_, _ = io.WriteString(h, frameName)
	u := h.Sum64() & 0x7fffffffffffffff
	return int64(u)
}

func buildStackRecords(
	weights ContainerFrameWeights,
	nodes map[int64]FrameNode,
	service, instanceType, hostname string,
	ts time.Time,
) []db.StackRecord {
	out := make([]db.StackRecord, 0, 4096)

	for rawContainer, w := range weights {
		containerName, k8sName := ContainerAndK8sName(rawContainer)
		for hash, samples := range w {
			node, ok := nodes[hash]
			if !ok {
				continue
			}
			out = append(out, db.StackRecord{
				Timestamp:        ts,
				Service:          service,
				InstanceType:     instanceType,
				ContainerEnvName: k8sName,
				HostName:         hostname,
				ContainerName:    containerName,
				NumSamples:       samples,
				CallStackHash:    hash,
				CallStackParent:  node.ParentHash,
				CallStackName:    node.Name,
			})
		}
	}
	return out
}

func buildMetricRecord(
	service, instanceType, hostname string,
	ts time.Time,
	cpuAvg, memAvg float64,
) *db.MetricRecord {
	if cpuAvg == 0 && memAvg == 0 {
		return nil
	}
	return &db.MetricRecord{
		Timestamp:                ts,
		Service:                  service,
		InstanceType:             instanceType,
		HostName:                 hostname,
		CPUAverageUsedPercent:    cpuAvg,
		MemoryAverageUsedPercent: memAvg,
	}
}
