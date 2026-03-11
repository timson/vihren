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

package db_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"vihren/internal/config"
	"vihren/internal/db"
	"vihren/internal/indexer"
	"vihren/internal/model/request"
)

const collapsedFile1 = `#{"metadata":{"hostname":"host-1","cloud_info":{"instance_type":"m5.large"},"run_arguments":{"service_name":"web-api","profile_api_version":"v2"}},"application_metadata_enabled":false}
app-container;main;handleRequest;processData 10
app-container;main;handleRequest;parseJSON 5
app-container;main;serveHTTP 3
`

const collapsedFile2 = `#{"metadata":{"hostname":"host-2","cloud_info":{"instance_type":"c5.xlarge"},"run_arguments":{"service_name":"web-api","profile_api_version":"v2"}},"application_metadata_enabled":false}
worker;main;handleRequest;processData 8
worker;main;handleRequest;sendResponse 4
`

const collapsedFile3 = `#{"metadata":{"hostname":"host-3","cloud_info":{"instance_type":"m5.large"},"run_arguments":{"service_name":"batch-job","profile_api_version":"v2"}},"application_metadata_enabled":false}
runner;main;runBatch;compute 20
`

// dbSink implements indexer.BlockSink by calling db.InsertStacks directly.
type dbSink struct {
	client *db.ChDBClient
}

func (s *dbSink) WriteProfileBlock(ctx context.Context, b db.ProfileBlock) error {
	return s.client.InsertStacks(ctx, b.Stacks)
}

func schemaPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "sql", "create_ch_schema.sql")
}

func newTestDB(t *testing.T) *db.ChDBClient {
	t.Helper()
	cfg := config.ChDBConfig{
		Filename:     filepath.Join(t.TempDir(), "testdb"),
		WriteBatchSize: 1,
		FlushEvery:   time.Second,
	}
	client, err := db.NewChDBClientWithSchema(cfg, schemaPath())
	if err != nil {
		t.Fatalf("newTestDB: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// ingestCollapsed runs the full ProfilesWriter pipeline: parse → aggregate → DB insert.
func ingestCollapsed(t *testing.T, client *db.ChDBClient, collapsed string, service string, ts time.Time) {
	t.Helper()
	writer := indexer.NewProfilesWriterWithSink(&dbSink{client: client}, nil)
	task := indexer.Task{
		Service:   service,
		Timestamp: ts,
	}
	if err := writer.ProcessStackFrameFileCtx(context.Background(), task, []byte(collapsed)); err != nil {
		t.Fatalf("ProcessStackFrameFileCtx: %v", err)
	}
}

func TestIntegration(t *testing.T) {
	ctx := context.Background()
	client := newTestDB(t)

	ts := time.Now().UTC().Truncate(time.Hour)

	ingestCollapsed(t, client, collapsedFile1, "web-api", ts)
	ingestCollapsed(t, client, collapsedFile2, "web-api", ts)
	ingestCollapsed(t, client, collapsedFile3, "batch-job", ts)

	// Force materialized view data to be visible.
	if err := client.Exec(ctx, "OPTIMIZE TABLE flamedb.samples FINAL"); err != nil {
		t.Fatalf("optimize samples: %v", err)
	}
	if err := client.Exec(ctx, "OPTIMIZE TABLE flamedb.samples_1min FINAL"); err != nil {
		t.Fatalf("optimize samples_1min: %v", err)
	}

	timeRange := request.TimeRangeQuery{
		StartTime: ts.Add(-time.Hour),
		EndTime:   ts.Add(time.Hour),
	}

	t.Run("RawQuery", func(t *testing.T) {
		params := request.FlameGraphQuery{
			TimeRangeQuery: timeRange,
			Service:        "web-api",
			StacksNum:      10000,
			Resolution:     "raw",
		}
		graph, err := client.GetTopFrames(ctx, params, "")
		if err != nil {
			t.Fatalf("GetTopFrames: %v", err)
		}
		var rootSamples int64
		for _, f := range graph.Frames {
			if f.IsRoot {
				rootSamples += f.Samples
			}
		}
		if rootSamples != 30 {
			t.Errorf("root samples = %d, want 30", rootSamples)
		}
	})

	t.Run("ServiceDiscovery", func(t *testing.T) {
		params := request.ServicesQuery{TimeRangeQuery: timeRange}
		services, err := client.FetchServices(ctx, params)
		if err != nil {
			t.Fatalf("FetchServices: %v", err)
		}
		sort.Strings(services)
		want := []string{"batch-job", "web-api"}
		if fmt.Sprintf("%v", services) != fmt.Sprintf("%v", want) {
			t.Errorf("services = %v, want %v", services, want)
		}
	})

	t.Run("FilterByHostname", func(t *testing.T) {
		params := request.FlameGraphQuery{
			TimeRangeQuery: timeRange,
			FiltersQuery:   request.FiltersQuery{HostName: []string{"host-1"}},
			Service:        "web-api",
			StacksNum:      10000,
			Resolution:     "raw",
		}
		graph, err := client.GetTopFrames(ctx, params, "")
		if err != nil {
			t.Fatalf("GetTopFrames: %v", err)
		}
		var rootSamples int64
		for _, f := range graph.Frames {
			if f.IsRoot {
				rootSamples += f.Samples
			}
		}
		if rootSamples != 18 {
			t.Errorf("root samples = %d, want 18", rootSamples)
		}
	})

	t.Run("FilterByInstanceType", func(t *testing.T) {
		params := request.FlameGraphQuery{
			TimeRangeQuery: timeRange,
			FiltersQuery:   request.FiltersQuery{InstanceType: []string{"c5.xlarge"}},
			Service:        "web-api",
			StacksNum:      10000,
			Resolution:     "raw",
		}
		graph, err := client.GetTopFrames(ctx, params, "")
		if err != nil {
			t.Fatalf("GetTopFrames: %v", err)
		}
		var rootSamples int64
		for _, f := range graph.Frames {
			if f.IsRoot {
				rootSamples += f.Samples
			}
		}
		if rootSamples != 12 {
			t.Errorf("root samples = %d, want 12", rootSamples)
		}
	})

	t.Run("FieldValues", func(t *testing.T) {
		params := request.QueryMetaQuery{
			TimeRangeQuery: timeRange,
			Service:        "web-api",
			LookupTarget:   "hostname",
		}
		values, err := client.FetchFieldValues(ctx, "HostName", params, "")
		if err != nil {
			t.Fatalf("FetchFieldValues: %v", err)
		}
		sort.Strings(values)
		want := []string{"host-1", "host-2"}
		if fmt.Sprintf("%v", values) != fmt.Sprintf("%v", want) {
			t.Errorf("values = %v, want %v", values, want)
		}
	})

	t.Run("SummaryStats", func(t *testing.T) {
		params := request.SummaryQuery{
			TimeRangeQuery: timeRange,
			Service:        "web-api",
		}
		stats, err := client.FetchSummaryStats(ctx, params, "")
		if err != nil {
			t.Fatalf("FetchSummaryStats: %v", err)
		}
		if stats.Samples != 30 {
			t.Errorf("samples = %d, want 30", stats.Samples)
		}
		if stats.Nodes != 2 {
			t.Errorf("nodes = %d, want 2", stats.Nodes)
		}
	})

	t.Run("SampleCount", func(t *testing.T) {
		params := request.QueryMetaQuery{
			TimeRangeQuery: timeRange,
			Service:        "web-api",
			LookupTarget:   "samples",
		}
		points, err := client.FetchSampleCount(ctx, params, "")
		if err != nil {
			t.Fatalf("FetchSampleCount: %v", err)
		}
		if len(points) == 0 {
			t.Fatal("expected non-empty sample points")
		}
		var total int64
		for _, p := range points {
			total += p.Samples
		}
		if total != 30 {
			t.Errorf("total samples = %d, want 30", total)
		}
	})

	t.Run("SessionsCount", func(t *testing.T) {
		params := request.SessionsCountQuery{
			TimeRangeQuery: timeRange,
			Service:        "web-api",
		}
		count, err := client.FetchSessionsCount(ctx, params, "")
		if err != nil {
			t.Fatalf("FetchSessionsCount: %v", err)
		}
		if count == 0 {
			t.Error("expected non-zero sessions count")
		}
	})

	t.Run("CrossServiceIsolation", func(t *testing.T) {
		params := request.FlameGraphQuery{
			TimeRangeQuery: timeRange,
			Service:        "batch-job",
			StacksNum:      10000,
			Resolution:     "raw",
		}
		graph, err := client.GetTopFrames(ctx, params, "")
		if err != nil {
			t.Fatalf("GetTopFrames: %v", err)
		}
		var rootSamples int64
		for _, f := range graph.Frames {
			if f.IsRoot {
				rootSamples += f.Samples
			}
		}
		if rootSamples != 20 {
			t.Errorf("root samples = %d, want 20", rootSamples)
		}
		for _, f := range graph.Frames {
			if f.Name == "handleRequest" || f.Name == "parseJSON" || f.Name == "serveHTTP" {
				t.Errorf("unexpected web-api frame %q in batch-job graph", f.Name)
			}
		}
	})
}
