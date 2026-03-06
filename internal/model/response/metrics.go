//
// Copyright (C) 2023 Intel Corporation
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

package response

import "time"

type SamplePoint struct {
	Time    time.Time `json:"time"`
	Samples int64     `json:"samples"`
}

type SampleCountByFunction struct {
	Time       time.Time `json:"time"`
	Percentage float64   `json:"cpu_percentage"`
}

type MetricsSummary struct {
	AvgCpu           float64    `json:"avg_cpu"`
	MaxCpu           float64    `json:"max_cpu"`
	AvgMemory        float64    `json:"avg_memory"`
	PercentileMemory float64    `json:"percentile_memory"`
	MaxMemory        float64    `json:"max_memory"`
	UniqHostnames    int        `json:"uniq_hostnames,omitempty"`
	GroupedBy        *string    `json:"grouped_by,omitempty"`
	Time             *time.Time `json:"time,omitempty"`
}

type SummaryStats struct {
	Samples       int64   `json:"samples"`
	AvgCpu        float64 `json:"avg_cpu"`
	MaxCpu        float64 `json:"max_cpu"`
	CurrentCpu    float64 `json:"current_cpu"`
	AvgMemory     float64 `json:"avg_memory"`
	MaxMemory     float64 `json:"max_memory"`
	CurrentMemory float64 `json:"current_memory"`
	Nodes         int     `json:"nodes"`
}

type MetricsCPUTrend struct {
	AvgCpu            float64 `json:"avg_cpu"`
	MaxCpu            float64 `json:"max_cpu"`
	AvgMemory         float64 `json:"avg_memory"`
	MaxMemory         float64 `json:"max_memory"`
	ComparedAvgCpu    float64 `json:"compared_avg_cpu"`
	ComparedMaxCpu    float64 `json:"compared_max_cpu"`
	ComparedAvgMemory float64 `json:"compared_avg_memory"`
	ComparedMaxMemory float64 `json:"compared_max_memory"`
}

type FilterValue struct {
	Name    string `json:"name"`
	Samples int64  `json:"samples,omitempty"`
}

type InstanceTypeCount struct {
	InstanceType  string `json:"instance_type"`
	InstanceCount int    `json:"instance_count"`
}

type ServiceMetricsSummary struct {
	MetricsSummary
	Service string `json:"service"`
}
