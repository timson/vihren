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

package response

import (
	"time"
	"vihren/internal/flamegraph"
)

type ExecTimeInterface interface {
	SetExecTime(time.Time)
}

type ExecTimeResponse struct {
	ExecTime float64 `json:"exec_time"`
}

func (et *ExecTimeResponse) SetExecTime(start time.Time) {
	et.ExecTime = float64(time.Since(start)) / float64(time.Second)
}

// Response is a generic wrapper for all standard API responses.
type Response[T any] struct {
	Result T `json:"result"`
	ExecTimeResponse
}

// FlameGraphResponse is kept separate because it carries additional top-level fields.
type FlameGraphResponse struct {
	Name        string                     `json:"name"`
	Value       int64                      `json:"value"`
	Children    []flamegraph.ResponseFrame `json:"children"`
	OlapTime    float64                    `json:"olap_time"`
	Percentiles map[string]string          `json:"percentiles"`
	ExecTimeResponse
}
