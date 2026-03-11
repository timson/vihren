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

package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"vihren/internal/config"

	_ "github.com/chdb-io/chdb-go/chdb/driver"
	log "github.com/sirupsen/logrus"
)

const (
	StacksTable      = "flamedb.samples_in"
	StacksQueryTable = "flamedb.samples"
	MetricsTable     = "flamedb.metrics"
	SchemaFile       = "sql/create_ch_schema.sql"
)

type ChDBClient struct {
	client *sql.DB
	config config.ChDBConfig
}

func NewChDBClient(cfg config.ChDBConfig) (*ChDBClient, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	return NewChDBClientWithSchema(cfg, filepath.Join(filepath.Dir(exePath), SchemaFile))
}

func NewChDBClientWithSchema(cfg config.ChDBConfig, schemaPath string) (*ChDBClient, error) {
	db, err := sql.Open("chdb", "session="+cfg.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open chdb: %w", err)
	}

	sqlSchema, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = db.Exec(string(sqlSchema))
	if err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	log.WithFields(log.Fields{
		"session": cfg.Filename,
		"schema":  schemaPath,
	}).Info("chdb init")

	return &ChDBClient{
		client: db,
		config: cfg,
	}, nil
}

func (c *ChDBClient) InsertStacks(ctx context.Context, rows []StackRecord) error {
	return insertStacks(ctx, c.client, rows)
}

func (c *ChDBClient) Close() error {
	return c.client.Close()
}

func (c *ChDBClient) Exec(ctx context.Context, query string) error {
	_, err := c.client.ExecContext(ctx, query)
	return err
}

func (c *ChDBClient) RunProfileWriter(ctx context.Context, in <-chan ProfileBlock) error {
	log.WithFields(log.Fields{
		"batch_size":  c.config.WriteBatchSize,
		"flush_every": c.config.FlushEvery,
	}).Debug("chdb writer start")

	t := time.NewTicker(c.config.FlushEvery)
	defer t.Stop()

	pendingStacks := make([]StackRecord, 0, c.config.WriteBatchSize)
	pendingMetrics := make([]MetricRecord, 0, 1024)

	flush := func() {
		if len(pendingStacks) == 0 && len(pendingMetrics) == 0 {
			return
		}

		stacksCount := len(pendingStacks)
		metricsCount := len(pendingMetrics)
		if len(pendingStacks) > 0 {
			if err := insertStacks(ctx, c.client, pendingStacks); err != nil {
				log.WithError(err).Error("chdb writer insert stacks failed")
			}
		}
		if len(pendingMetrics) > 0 {
			if err := insertMetrics(ctx, c.client, pendingMetrics); err != nil {
				log.WithError(err).Error("chdb writer insert metrics failed")
			}
		}

		pendingStacks = pendingStacks[:0]
		pendingMetrics = pendingMetrics[:0]
		log.WithFields(log.Fields{
			"stacks":  stacksCount,
			"metrics": metricsCount,
		}).Debug("chdb writer flushed")
	}

	for {
		select {
		case <-ctx.Done():
			log.WithError(ctx.Err()).Debug("chdb writer ctx done")
			flush()
			return ctx.Err()

		case <-t.C:
			flush()

		case pb, ok := <-in:
			if !ok {
				log.Debug("chdb writer input closed")
				flush()
				return nil
			}

			log.WithFields(log.Fields{
				"stacks":  len(pb.Stacks),
				"metrics": pb.Metrics != nil,
			}).Debug("chdb writer recv")
			if len(pb.Stacks) > 0 {
				pendingStacks = append(pendingStacks, pb.Stacks...)
			}
			if pb.Metrics != nil {
				pendingMetrics = append(pendingMetrics, *pb.Metrics)
			}

			if len(pendingStacks) >= c.config.WriteBatchSize {
				flush()
			}
		}
	}
}

func insertStacks(ctx context.Context, db *sql.DB, rows []StackRecord) error {
	if len(rows) == 0 {
		return nil
	}

	const cols = "(Timestamp, Service, InstanceType, ContainerEnvName, HostName, ContainerName, NumSamples, CallStackHash, CallStackName, CallStackParent)"
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "INSERT INTO %s %s VALUES ", StacksTable, cols)

	args := make([]any, 0, len(rows)*11)
	for i, r := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		ts := r.Timestamp.UTC().Truncate(time.Second)
		b.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			ts,
			r.Service,
			r.InstanceType,
			r.ContainerEnvName,
			r.HostName,
			r.ContainerName,
			r.NumSamples,
			r.CallStackHash,
			r.CallStackName,
			r.CallStackParent,
		)
	}

	_, err := db.ExecContext(ctx, b.String(), args...)
	return err
}

func insertMetrics(ctx context.Context, db *sql.DB, rows []MetricRecord) error {
	if len(rows) == 0 {
		return nil
	}

	const cols = "(Timestamp, Service, InstanceType, HostName, CPUAverageUsedPercent, MemoryAverageUsedPercent)"
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "INSERT INTO %s %s VALUES ", MetricsTable, cols)

	args := make([]any, 0, len(rows)*6)
	for i, r := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		ts := r.Timestamp.UTC().Truncate(time.Second)
		b.WriteString("(?, ?, ?, ?, ?, ?)")
		args = append(args,
			ts,
			r.Service,
			r.InstanceType,
			r.HostName,
			r.CPUAverageUsedPercent,
			r.MemoryAverageUsedPercent,
		)
	}

	_, err := db.ExecContext(ctx, b.String(), args...)
	return err
}
