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

package request

import (
	"time"
)

const (
	BackRewindTime = 2 * time.Minute
)

type TimeRangeQuery struct {
	StartTime time.Time `form:"start_datetime" time_format:"2006-01-02T15:04:05" time_utc:"1"`
	EndTime   time.Time `form:"end_datetime" time_format:"2006-01-02T15:04:05" time_utc:"1"`
}

func (p *TimeRangeQuery) NormalizeTimeRange() {
	now := time.Now().UTC().Truncate(time.Second)

	if p.EndTime.IsZero() {
		p.EndTime = now.Add(-BackRewindTime)
	} else {
		p.EndTime = p.EndTime.UTC().Truncate(time.Second)
	}

	if p.StartTime.IsZero() {
		p.StartTime = p.EndTime.Add(-24 * time.Hour)
	} else {
		p.StartTime = p.StartTime.UTC().Truncate(time.Second)
	}

	if p.StartTime.After(p.EndTime) {
		p.StartTime, p.EndTime = p.EndTime, p.StartTime
	}
}

type FiltersQuery struct {
	ContainerName []string `form:"container"`
	HostName      []string `form:"hostname"`
	InstanceType  []string `form:"instance_type"`
	Workload      []string `form:"workload"`
}

type RQLFilters struct {
	ContainerName string `rql:"column=ContainerName,filter"`
	HostName      string `rql:"column=HostName,filter"`
	InstanceType  string `rql:"column=InstanceType,filter"`
	Workload      string `rql:"column=ContainerEnvName,filter"`
}

type MetricsRQLFilters struct {
	HostName     string `rql:"column=HostName,filter"`
	InstanceType string `rql:"column=InstanceType,filter"`
}

type FlameGraphQuery struct {
	TimeRangeQuery
	FiltersQuery
	Service    string `form:"service,default=0"`
	Filter     string `form:"filter"`
	StacksNum  int    `form:"stacks_num,default=10000"`
	Sample     int    `form:"sample,default=1"`
	Resolution string `form:"resolution,default=multi" binding:"oneof=multi hour day raw"`
	Format     string `form:"format,default=flamegraph" binding:"oneof=flamegraph collapsed_file svg"`
}

func (p *FlameGraphQuery) FilterQuery() string {
	return p.Filter
}

type QueryMetaQuery struct {
	TimeRangeQuery
	FiltersQuery
	Service      string `form:"service" binding:"required"`
	FunctionName string `form:"function_name"`
	Filter       string `form:"filter"`
	Resolution   string `form:"resolution,default=hour" binding:"oneof=none hour day raw"`
	Interval     string `form:"interval"`
	LookupTarget string `form:"lookup_for" binding:"required,oneof=container hostname instance_type workload time time_range instance_type_count samples"`
}

func (p *QueryMetaQuery) FilterQuery() string {
	return p.Filter
}

type ServicesQuery struct {
	TimeRangeQuery
	WithDeployments bool `form:"with_deployments,default=false"`
}

type SessionsCountQuery struct {
	TimeRangeQuery
	FiltersQuery
	Service string `form:"service" binding:"required"`
	Filter  string `form:"filter"`
}

func (p *SessionsCountQuery) FilterQuery() string {
	return p.Filter
}

type MetricsFiltersQuery struct {
	HostName     []string `form:"hostname"`
	InstanceType []string `form:"instance_type"`
}

type MetricsSummaryQuery struct {
	TimeRangeQuery
	MetricsFiltersQuery
	Service    string `form:"service" binding:"required"`
	Filter     string `form:"filter"`
	Percentile int    `form:"percentile,default=90" binding:"numeric,min=0,max=100"`
	Interval   string `form:"interval"`
}

type SummaryQuery struct {
	TimeRangeQuery
	FiltersQuery
	Service string `form:"service" binding:"required"`
	Filter  string `form:"filter"`
}

func (p *SummaryQuery) FilterQuery() string {
	return p.Filter
}

func (p *MetricsSummaryQuery) FilterQuery() string {
	return p.Filter
}
