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

package config

import (
	"time"
)

type ServerConfig struct {
	Port int `default:"8080"`
}

type QueueConfig struct {
	PayloadPath string        `default:"/tmp/profiles"`
	QueueDepth  int           `default:"100"`
	SendTimeout time.Duration `default:"10s"`
}

type ChDBConfig struct {
	Filename            string        `default:"flamedb"`
	RawRetentionDays    int           `default:"7"`   // Raw data retention period
	MinuteRetentionDays int           `default:"365"` // Minute aggregation retention period
	HourlyRetentionDays int           `default:"90"`  // Hourly aggregation retention period
	DailyRetentionDays  int           `default:"365"` // Daily aggregation retention period
	MinStackRows        int           `default:"100_000"`
	FlushEvery          time.Duration `default:"10s"`
}

type IndexerConfig struct {
	Workers        int    `default:"2"`
	NormalizerPath string `default:"replace.yaml"`
}

type VihrenConfig struct {
	Queue   QueueConfig
	Server  ServerConfig
	DB      ChDBConfig
	Indexer IndexerConfig
}
