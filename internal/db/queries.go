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

package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"vihren/internal/flamegraph"
	"vihren/internal/model/request"
	"vihren/internal/model/response"
	"vihren/internal/stackframe"

	sq "github.com/Masterminds/squirrel"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	day                            = time.Hour * 24
	defaultStep                    = "1 minute"
	frameQueryChunkLimitMultiplier = 3
	frameHashBatchSize             = 512
	maxAncestorExpansionIters      = 16
)


func resolveInterval(start, end time.Time, interval string) string {
	if interval != "" {
		return interval
	}
	diff := end.Sub(start)
	if diff <= 0 {
		return "1 minute"
	}

	const maxBuckets = int64(180)
	target := time.Duration(int64(diff) / maxBuckets)
	if target <= 0 {
		target = time.Second
	}

	type cand struct {
		d   time.Duration
		txt string
	}
	cands := []cand{
		{1 * time.Second, "1 second"}, {2 * time.Second, "2 second"}, {5 * time.Second, "5 second"},
		{10 * time.Second, "10 second"}, {15 * time.Second, "15 second"}, {30 * time.Second, "30 second"},
		{1 * time.Minute, "1 minute"}, {2 * time.Minute, "2 minute"}, {5 * time.Minute, "5 minute"},
		{10 * time.Minute, "10 minute"}, {15 * time.Minute, "15 minute"}, {30 * time.Minute, "30 minute"},
		{1 * time.Hour, "1 hour"}, {2 * time.Hour, "2 hour"}, {3 * time.Hour, "3 hour"},
		{6 * time.Hour, "6 hour"}, {12 * time.Hour, "12 hour"},
		{24 * time.Hour, "1 day"}, {48 * time.Hour, "2 day"}, {7 * 24 * time.Hour, "7 day"},
	}

	for _, c := range cands {
		if c.d >= target {
			return c.txt
		}
	}
	return cands[len(cands)-1].txt
}

func BuildConditionsSq(p *request.FiltersQuery, filterQuery string) (tablePrefix string, where sq.Sqlizer) {
	if filterQuery != "" {
		return "", sq.Expr(filterQuery) // raw override
	}

	tablePrefix = "_all"
	parts := make([]sq.Sqlizer, 0, 4)

	if len(p.ContainerName) > 0 {
		tablePrefix = ""
		parts = append(parts, sq.Eq{"ContainerName": p.ContainerName})
	}
	if len(p.HostName) > 0 {
		tablePrefix = ""
		parts = append(parts, sq.Eq{"HostName": p.HostName})
	}
	if len(p.InstanceType) > 0 {
		tablePrefix = ""
		parts = append(parts, sq.Eq{"InstanceType": p.InstanceType})
	}
	if len(p.K8SObject) > 0 {
		tablePrefix = ""
		parts = append(parts, sq.Eq{"ContainerEnvName": p.K8SObject})
	}

	if len(parts) == 0 {
		return tablePrefix, nil
	}
	return tablePrefix, sq.And(parts)
}

func buildMetricsConditions(p *request.FiltersQuery, filterQuery string) sq.Sqlizer {
	if filterQuery != "" &&
		!strings.Contains(filterQuery, "ContainerName") &&
		!strings.Contains(filterQuery, "ContainerEnvName") {
		return sq.Expr(filterQuery)
	}
	if p == nil {
		return nil
	}
	parts := make([]sq.Sqlizer, 0, 2)
	if len(p.HostName) > 0 {
		parts = append(parts, sq.Eq{"HostName": p.HostName})
	}
	if len(p.InstanceType) > 0 {
		parts = append(parts, sq.Eq{"InstanceType": p.InstanceType})
	}
	if len(parts) == 0 {
		return nil
	}
	return sq.And(parts)
}

func getTableName(table string, tablePrefix string) string {
	if table == "raw" {
		return StacksQueryTable
	}
	if table == "1day_historical" {
		return fmt.Sprintf("%s_1day", StacksQueryTable)
	}
	return fmt.Sprintf("%s_%s%s", StacksQueryTable, table, tablePrefix)
}

type framePair struct {
	Hash       int64
	ParentHash int64
	Samples    int64
}

type frameAggregate struct {
	Samples       int64
	ParentSamples map[int64]int64
}

func scanFramePairs(rows *sql.Rows) ([]framePair, error) {
	pairs := make([]framePair, 0, 1024)
	for rows.Next() {
		var hash, parentHash int64
		var samples int64
		err := rows.Scan(&hash, &parentHash, &samples)
		if err != nil {
			log.WithError(err).Error("scan frame pair row failed")
			continue
		}
		if samples == 0 {
			continue
		}
		pairs = append(pairs, framePair{Hash: hash, ParentHash: parentHash, Samples: samples})
	}
	return pairs, rows.Err()
}

func mergeFramePairs(dst map[int64]*frameAggregate, pairs []framePair) {
	for _, p := range pairs {
		agg, ok := dst[p.Hash]
		if !ok {
			agg = &frameAggregate{ParentSamples: make(map[int64]int64)}
			dst[p.Hash] = agg
		}
		agg.Samples += p.Samples
		agg.ParentSamples[p.ParentHash] += p.Samples
	}
}

func mergeFrameAggregates(dst map[int64]*frameAggregate, src map[int64]*frameAggregate) {
	for hash, srcAgg := range src {
		dstAgg, ok := dst[hash]
		if !ok {
			dstAgg = &frameAggregate{ParentSamples: make(map[int64]int64)}
			dst[hash] = dstAgg
		}
		dstAgg.Samples += srcAgg.Samples
		for parentHash, samples := range srcAgg.ParentSamples {
			dstAgg.ParentSamples[parentHash] += samples
		}
	}
}

func stableParentHash(parentWeights map[int64]int64) int64 {
	if len(parentWeights) == 0 {
		return 0
	}
	var parent int64
	var maxSamples int64 = -1
	first := true
	for candidate, samples := range parentWeights {
		if first || samples > maxSamples || (samples == maxSamples && candidate < parent) {
			parent = candidate
			maxSamples = samples
			first = false
		}
	}
	return parent
}

func topFrameHashes(aggregates map[int64]*frameAggregate, n int) []int64 {
	if n <= 0 || len(aggregates) == 0 {
		return nil
	}
	hashes := make([]int64, 0, len(aggregates))
	for hash := range aggregates {
		hashes = append(hashes, hash)
	}
	sort.Slice(hashes, func(i, j int) bool {
		si := aggregates[hashes[i]].Samples
		sj := aggregates[hashes[j]].Samples
		if si == sj {
			return hashes[i] < hashes[j]
		}
		return si > sj
	})
	if n > len(hashes) {
		n = len(hashes)
	}
	return hashes[:n]
}

func hashSet(hashes []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(hashes))
	for _, hash := range hashes {
		out[hash] = struct{}{}
	}
	return out
}

func sortedHashes(set map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(set))
	for hash := range set {
		out = append(out, hash)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func expandKnownAncestors(selected map[int64]struct{}, aggregates map[int64]*frameAggregate) bool {
	changed := false
	for {
		expanded := false
		for hash := range selected {
			agg, ok := aggregates[hash]
			if !ok {
				continue
			}
			parentHash := stableParentHash(agg.ParentSamples)
			if parentHash == 0 {
				continue
			}
			if _, ok := selected[parentHash]; ok {
				continue
			}
			if _, ok := aggregates[parentHash]; !ok {
				continue
			}
			selected[parentHash] = struct{}{}
			expanded = true
			changed = true
		}
		if !expanded {
			return changed
		}
	}
}

func missingAncestors(selected map[int64]struct{}, aggregates map[int64]*frameAggregate) []int64 {
	missing := make(map[int64]struct{})
	for hash := range selected {
		agg, ok := aggregates[hash]
		if !ok {
			continue
		}
		parentHash := stableParentHash(agg.ParentSamples)
		if parentHash == 0 {
			continue
		}
		if _, ok := selected[parentHash]; ok {
			continue
		}
		if _, ok := aggregates[parentHash]; ok {
			continue
		}
		missing[parentHash] = struct{}{}
	}
	return sortedHashes(missing)
}

func buildFramesFromAggregates(aggregates map[int64]*frameAggregate, selected map[int64]struct{}) map[int64]stackframe.Frame {
	frames := make(map[int64]stackframe.Frame, len(selected))
	for hash := range selected {
		agg, ok := aggregates[hash]
		if !ok {
			continue
		}
		parentHash := stableParentHash(agg.ParentSamples)
		frames[hash] = stackframe.Frame{
			Hash:       hash,
			ParentHash: parentHash,
			Name:       "",
			Children:   make(map[int64]bool),
			Samples:    agg.Samples,
			IsRoot:     parentHash == 0,
		}
	}
	return frames
}

func calcFrameChunkLimit(stacksNum int) int {
	stacksNum = effectiveStacksNum(stacksNum)
	return stacksNum * frameQueryChunkLimitMultiplier
}

func effectiveStacksNum(stacksNum int) int {
	if stacksNum <= 0 {
		return 10_000
	}
	return stacksNum
}

func frameNamesTimeRange(start, end time.Time) (time.Time, time.Time) {
	startDay := start.Truncate(day)
	endDay := end.Truncate(day)
	if !end.Equal(endDay) {
		endDay = endDay.Add(day)
	}
	if !startDay.Before(endDay) {
		endDay = startDay.Add(day)
	}
	return startDay, endDay
}

func splitHashBatches(hashes []int64, size int) [][]int64 {
	if len(hashes) == 0 {
		return nil
	}
	if size <= 0 {
		size = len(hashes)
	}
	out := make([][]int64, 0, (len(hashes)+size-1)/size)
	for start := 0; start < len(hashes); start += size {
		end := start + size
		if end > len(hashes) {
			end = len(hashes)
		}
		batch := make([]int64, end-start)
		copy(batch, hashes[start:end])
		out = append(out, batch)
	}
	return out
}

func (c *ChDBClient) queryFramePairs(
	ctx context.Context,
	entry *log.Entry,
	tableName string,
	service string,
	tr Chunk,
	conds sq.Sqlizer,
	limit int,
	hashes []int64,
) ([]framePair, error) {
	qb := sq.
		Select("CallStackHash", "CallStackParent", "sum(NumSamples) AS SumNumSamples").
		From(tableName).
		Where(sq.Eq{"Service": service}).
		Where("Timestamp >= ? AND Timestamp < ?", tr.Start, tr.End).
		GroupBy("CallStackHash, CallStackParent")

	if len(hashes) > 0 {
		qb = qb.Where(sq.Eq{"CallStackHash": hashes})
	}
	if conds != nil {
		qb = qb.Where(conds)
	}
	if limit > 0 {
		qb = qb.OrderBy("SumNumSamples DESC").Limit(uint64(limit))
	}

	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}
	return QueryRows[[]framePair](ctx, c.client, entry, query, args, scanFramePairs)
}

func (c *ChDBClient) fetchFramePairsByHashes(
	ctx context.Context,
	params request.FlameGraphQuery,
	timeRanges []Chunk,
	tablePrefix string,
	conds sq.Sqlizer,
	hashes []int64,
) (map[int64]*frameAggregate, error) {
	aggregates := make(map[int64]*frameAggregate)
	if len(hashes) == 0 {
		return aggregates, nil
	}

	batches := splitHashBatches(hashes, frameHashBatchSize)
	var eg errgroup.Group
	var mu sync.Mutex

	for _, timeRange := range timeRanges {
		tr := timeRange
		tableName := getTableName(tr.Table, tablePrefix)
		for _, hashBatch := range batches {
			batch := hashBatch
			entry := log.WithFields(log.Fields{
				"service": params.Service,
				"table":   tableName,
				"start":   tr.Start,
				"end":     tr.End,
				"hashes":  len(batch),
			})
			eg.Go(func() error {
				pairs, err := c.queryFramePairs(ctx, entry, tableName, params.Service, tr, conds, 0, batch)
				if err != nil {
					return err
				}
				if len(pairs) == 0 {
					return nil
				}
				mu.Lock()
				mergeFramePairs(aggregates, pairs)
				mu.Unlock()
				return nil
			})
		}
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return aggregates, nil
}

func scanFrameNames(rows *sql.Rows) (map[int64]string, error) {
	names := make(map[int64]string)
	for rows.Next() {
		var hash int64
		var name string
		if err := rows.Scan(&hash, &name); err != nil {
			log.WithError(err).Error("scan frame name row failed")
			continue
		}
		names[hash] = name
	}
	return names, rows.Err()
}

func (c *ChDBClient) fetchFrameNames(
	ctx context.Context,
	service string,
	hashes []int64,
	start time.Time,
	end time.Time,
) (map[int64]string, error) {
	names := make(map[int64]string)
	if len(hashes) == 0 {
		return names, nil
	}

	startDay, endDay := frameNamesTimeRange(start, end)
	batches := splitHashBatches(hashes, frameHashBatchSize)
	entry := log.WithFields(log.Fields{"service": service, "table": "flamedb.samples_name"})

	for _, hashBatch := range batches {
		qb := sq.
			Select("CallStackHash", "any(CallStackName) AS CallStackName").
			From("flamedb.samples_name").
			Where(sq.Eq{"Service": service}).
			Where("Timestamp >= ? AND Timestamp < ?", startDay, endDay).
			Where(sq.Eq{"CallStackHash": hashBatch}).
			GroupBy("CallStackHash").
			Suffix("SETTINGS max_query_size=4194304")

		query, args, err := qb.ToSql()
		if err != nil {
			return nil, err
		}

		batchNames, err := QueryRows[map[int64]string](ctx, c.client, entry, query, args, scanFrameNames)
		if err != nil {
			return nil, err
		}
		for hash, name := range batchNames {
			names[hash] = name
		}
	}
	return names, nil
}

func (c *ChDBClient) GetTopFrames(ctx context.Context, params request.FlameGraphQuery, filterQuery string) (*flamegraph.Graph, error) {
	var eg errgroup.Group
	var mu sync.Mutex

	graph := flamegraph.NewGraph()
	timeRanges, err := BuildChunks(params.StartTime, params.EndTime, params.Resolution)
	if err != nil {
		return nil, err
	}
	tablePrefix, conds := BuildConditionsSq(&params.FiltersQuery, filterQuery)
	chunkLimit := calcFrameChunkLimit(params.StacksNum)
	aggregates := make(map[int64]*frameAggregate)

	for _, timeRange := range timeRanges {
		tr := timeRange
		tableName := getTableName(tr.Table, tablePrefix)
		entry := log.WithFields(log.Fields{
			"service": params.Service,
			"table":   tableName,
			"start":   tr.Start,
			"end":     tr.End,
			"limit":   chunkLimit,
		})

		eg.Go(func() error {
			pairs, queryErr := c.queryFramePairs(ctx, entry, tableName, params.Service, tr, conds, chunkLimit, nil)
			if queryErr != nil {
				return queryErr
			}
			if len(pairs) == 0 {
				return nil
			}
			mu.Lock()
			mergeFramePairs(aggregates, pairs)
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("unable fetch flamegraph from DB: %w", err)
	}

	if len(aggregates) > 0 {
		conflicts := 0
		for _, agg := range aggregates {
			if len(agg.ParentSamples) > 1 {
				conflicts++
			}
		}
		if conflicts > 0 {
			log.WithFields(log.Fields{
				"service":         params.Service,
				"conflict_hashes": conflicts,
			}).Warn("multiple parent candidates detected for frame hashes")
		}

		selected := hashSet(topFrameHashes(aggregates, effectiveStacksNum(params.StacksNum)))
		for iter := 0; iter < maxAncestorExpansionIters; iter++ {
			expanded := expandKnownAncestors(selected, aggregates)
			missing := missingAncestors(selected, aggregates)
			if len(missing) == 0 {
				if !expanded {
					break
				}
				continue
			}

			fetched, fetchErr := c.fetchFramePairsByHashes(ctx, params, timeRanges, tablePrefix, conds, missing)
			if fetchErr != nil {
				return nil, fmt.Errorf("unable fetch flamegraph ancestors: %w", fetchErr)
			}
			if len(fetched) == 0 {
				log.WithFields(log.Fields{
					"service":           params.Service,
					"missing_ancestors": len(missing),
				}).Warn("unable to resolve all missing ancestors for top frames")
				break
			}
			mergeFrameAggregates(aggregates, fetched)
		}
		expandKnownAncestors(selected, aggregates)
		graph.UpdateFrames(buildFramesFromAggregates(aggregates, selected))
	}

	if len(graph.Frames) > 0 {
		hashes := make([]int64, 0, len(graph.Frames))
		for hash := range graph.Frames {
			hashes = append(hashes, hash)
		}

		names, err := c.fetchFrameNames(ctx, params.Service, hashes, params.StartTime, params.EndTime)
		if err != nil {
			return nil, err
		}

		for hash, name := range names {
			frame := graph.Frames[hash]
			frame.Name = name
			graph.Frames[hash] = frame
		}
	}

	_, err = graph.PrepareFrames(0)
	if err != nil {
		return nil, err
	}

	return &graph, nil
}

func (c *ChDBClient) FetchInstanceTypeCount(ctx context.Context, params request.QueryMetaQuery,
	filterQuery string) ([]response.InstanceTypeCount, error) {
	entry := log.WithField("service", params.Service)
	query := sq.Select("InstanceType, COUNT(DISTINCT HostName) as InstanceCount").
		From("flamedb.samples_1min").
		Where(sq.Eq{"Service": params.Service}).
		Where("InstanceType != ''").
		Where("Timestamp BETWEEN ? AND ?", params.StartTime, params.EndTime).
		GroupBy("InstanceType").
		OrderBy("InstanceCount DESC")

	_, query = ApplyParams(query, &params.FiltersQuery, filterQuery)

	return SelectAllSq[response.InstanceTypeCount](ctx, c.client, entry, query, func(rows *sql.Rows) (response.InstanceTypeCount, error) {
		result := response.InstanceTypeCount{}
		err := rows.Scan(&result.InstanceType, &result.InstanceCount)
		return result, err
	})
}

func (c *ChDBClient) FetchFieldValues(ctx context.Context, field string, params request.QueryMetaQuery,
	filterQuery string) ([]string, error) {
	entry := log.WithFields(log.Fields{
		"service": params.Service,
		"field":   field,
	})
	qb := sq.Select(field).From("flamedb.samples_1min").Where(sq.Eq{"Service": params.Service}).
		Where("Timestamp BETWEEN ? AND ?", params.StartTime, params.EndTime).
		GroupBy(field)

	entry.Debug(qb.ToSql())

	_, qb = ApplyParams(qb, &params.FiltersQuery, filterQuery)

	return SelectAllSq[string](ctx, c.client, entry, qb, func(r *sql.Rows) (string, error) {
		var s string
		return s, r.Scan(&s)
	})
}

func (c *ChDBClient) FetchSampleCount(ctx context.Context, params request.QueryMetaQuery,
	filterQuery string) ([]response.SamplePoint, error) {
	interval := resolveInterval(params.StartTime, params.EndTime, params.Interval)
	entry := log.WithField("service", params.Service)
	query := sq.Select(fmt.Sprintf("toStartOfInterval(Timestamp, INTERVAL %s) as Datetime, SUM(NumSamples)", interval)).
		From("flamedb.samples_1min").
		Where(sq.Eq{"Service": params.Service}).
		Where("Timestamp BETWEEN ? AND ?", params.StartTime, params.EndTime).
		GroupBy("Datetime").
		OrderBy("Datetime ASC").
		Suffix(fmt.Sprintf("WITH FILL STEP INTERVAL %s", interval))
	_, query = ApplyParams(query, &params.FiltersQuery, filterQuery)

	return SelectAllSq[response.SamplePoint](ctx, c.client, entry, query, func(rows *sql.Rows) (response.SamplePoint, error) {
		sp := response.SamplePoint{}
		err := rows.Scan(&sp.Time, &sp.Samples)
		return sp, err
	})
}

func (c *ChDBClient) FetchMetricsGraph(ctx context.Context, params request.MetricsSummaryQuery,
	filterQuery string) ([]response.MetricsSummary, error) {
	entry := log.WithField("service", params.Service)
	interval := resolveInterval(params.StartTime, params.EndTime, params.Interval)
	percentile := float64(params.Percentile) / 100.0

	inner := sq.
		Select(
			fmt.Sprintf("toStartOfInterval(Timestamp, INTERVAL %s) AS Datetime", interval),
			"MAX(MemoryAverageUsedPercent) AS MaxMemory",
			"MAX(CPUAverageUsedPercent) AS MaxCPU",
			"groupArray(CPUAverageUsedPercent) AS CPUArray",
		).
		From(MetricsTable).
		Where(sq.Eq{"Service": params.Service}).
		Where("Timestamp BETWEEN ? AND ?", params.StartTime, params.EndTime).
		GroupBy("Datetime").
		OrderBy("Datetime ASC").
		Suffix(fmt.Sprintf("WITH FILL STEP INTERVAL %s INTERPOLATE(MaxMemory, MaxCPU, CPUArray)", interval))

	query := sq.
		Select(
			"Datetime",
			"arrayAvg(flatten(groupArray(CPUArray)))",
			"MAX(MaxCPU)",
			"AVG(MaxMemory)",
			"MAX(MaxMemory)",
			fmt.Sprintf("quantile(%f)(MaxMemory)", percentile),
		).
		FromSelect(inner, "t").
		GroupBy("Datetime")

	_, query = ApplyParams(query, &params.FiltersQuery, filterQuery)

	return SelectAllSq[response.MetricsSummary](ctx, c.client, entry, query, func(rows *sql.Rows) (response.MetricsSummary, error) {
		ms := response.MetricsSummary{}
		err := rows.Scan(&ms.Time, &ms.AvgCpu, &ms.MaxCpu, &ms.AvgMemory, &ms.MaxMemory, &ms.PercentileMemory)
		return ms, err
	})
}

func (c *ChDBClient) FetchSummaryStats(ctx context.Context, params request.SummaryQuery,
	filterQuery string) (response.SummaryStats, error) {
	entry := log.WithField("service", params.Service)

	samplesQuery := sq.
		Select(
			"ifNull(sum(NumSamples), 0) AS Samples",
			"ifNull(uniq(HostName), 0) AS Nodes",
		).
		From("flamedb.samples_1min").
		Where(sq.Eq{"Service": params.Service}).
		Where("Timestamp BETWEEN ? AND ?", params.StartTime, params.EndTime)
	_, samplesQuery = ApplyParams(samplesQuery, &params.FiltersQuery, filterQuery)

	metricsQuery := sq.
		Select(
			"ifNull(avg(CPUAverageUsedPercent), 0) AS AvgCpu",
			"ifNull(max(CPUAverageUsedPercent), 0) AS MaxCpu",
			"ifNull(argMax(CPUAverageUsedPercent, Timestamp), 0) AS CurrentCpu",
			"ifNull(avg(MemoryAverageUsedPercent), 0) AS AvgMemory",
			"ifNull(max(MemoryAverageUsedPercent), 0) AS MaxMemory",
			"ifNull(argMax(MemoryAverageUsedPercent, Timestamp), 0) AS CurrentMemory",
		).
		From(MetricsTable).
		Where(sq.Eq{"Service": params.Service}).
		Where("Timestamp BETWEEN ? AND ?", params.StartTime, params.EndTime)
	if metricsWhere := buildMetricsConditions(&params.FiltersQuery, filterQuery); metricsWhere != nil {
		metricsQuery = metricsQuery.Where(metricsWhere)
	}

	var samplesAgg, metricsAgg response.SummaryStats
	var eg errgroup.Group

	eg.Go(func() error {
		var err error
		samplesAgg, err = SelectOneSq[response.SummaryStats](ctx, c.client, entry, samplesQuery, func(rows *sql.Rows) (response.SummaryStats, error) {
			out := response.SummaryStats{}
			return out, rows.Scan(&out.Samples, &out.Nodes)
		})
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	})

	eg.Go(func() error {
		var err error
		metricsAgg, err = SelectOneSq[response.SummaryStats](ctx, c.client, entry, metricsQuery, func(rows *sql.Rows) (response.SummaryStats, error) {
			out := response.SummaryStats{}
			return out, rows.Scan(&out.AvgCpu, &out.MaxCpu, &out.CurrentCpu, &out.AvgMemory, &out.MaxMemory, &out.CurrentMemory)
		})
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	})

	if err := eg.Wait(); err != nil {
		return response.SummaryStats{}, err
	}

	return response.SummaryStats{
		Samples:       samplesAgg.Samples,
		Nodes:         samplesAgg.Nodes,
		AvgCpu:        metricsAgg.AvgCpu,
		MaxCpu:        metricsAgg.MaxCpu,
		CurrentCpu:    metricsAgg.CurrentCpu,
		AvgMemory:     metricsAgg.AvgMemory,
		MaxMemory:     metricsAgg.MaxMemory,
		CurrentMemory: metricsAgg.CurrentMemory,
	}, nil
}

func (c *ChDBClient) FetchServices(ctx context.Context, params request.ServicesQuery) ([]string, error) {
	entry := log.WithFields(log.Fields{"start": params.StartTime, "end": params.EndTime})
	qb := sq.Select("DISTINCT Service").From("flamedb.samples_1min").Where("Timestamp BETWEEN ? AND ?", params.StartTime, params.EndTime)
	return SelectAllSq[string](ctx, c.client, entry, qb, func(r *sql.Rows) (string, error) {
		var s string
		return s, r.Scan(&s)
	})
}

func (c *ChDBClient) FetchSessionsCount(ctx context.Context, params request.SessionsCountQuery,
	filterQuery string) (int, error) {
	entry := log.WithFields(log.Fields{"start": params.StartTime, "end": params.EndTime, "service": params.Service})
	qb := sq.
		Select("uniq(HostName, Timestamp)").
		From("flamedb.samples_1min").
		Where(sq.Eq{"Service": params.Service}).
		Where("Timestamp BETWEEN ? AND ?", params.StartTime, params.EndTime)

	_, qb = ApplyParams(qb, &params.FiltersQuery, filterQuery)

	v, err := SelectOneSq[int](ctx, c.client, entry, qb, func(r *sql.Rows) (int, error) {
		var n int
		return n, r.Scan(&n)
	})

	if err != nil {
		return 0, err
	}
	return v, nil
}

func (c *ChDBClient) FetchDBStatus(ctx context.Context) (response.DBStatus, error) {
	entry := log.WithField("db", "flamedb")
	type tableRow struct {
		Name                  string
		Rows                  sql.NullInt64
		BytesOnDisk           sql.NullInt64
		DataUncompressedBytes sql.NullInt64
	}

	tableQuery := sq.Select(
		"name",
		"ifNull(total_rows, 0)",
		"ifNull(total_bytes, 0)",
		"ifNull(total_bytes_uncompressed, 0)",
	).
		From("system.tables").
		Where(sq.Eq{"database": "flamedb"}).
		Where("engine NOT IN ('MaterializedView', 'Null')").
		OrderBy("name ASC")

	tables, err := SelectAllSq[tableRow](ctx, c.client, entry, tableQuery, func(rows *sql.Rows) (tableRow, error) {
		var row tableRow
		err := rows.Scan(&row.Name, &row.Rows, &row.BytesOnDisk, &row.DataUncompressedBytes)
		return row, err
	})
	if err != nil {
		return response.DBStatus{}, err
	}

	timestampTables := make(map[string]struct{})
	columnsQuery := sq.Select("table").
		From("system.columns").
		Where(sq.Eq{"database": "flamedb", "name": "Timestamp"}).
		GroupBy("table")

	columns, err := SelectAllSq[string](ctx, c.client, entry, columnsQuery, func(rows *sql.Rows) (string, error) {
		var name string
		return name, rows.Scan(&name)
	})
	if err == nil {
		for _, name := range columns {
			timestampTables[name] = struct{}{}
		}
	}

	status := response.DBStatus{
		Tables: make([]response.TableStatus, 0, len(tables)),
	}

	for _, tbl := range tables {
		rows := int64(0)
		bytesOnDisk := int64(0)
		dataUncompressed := int64(0)
		if tbl.Rows.Valid {
			rows = tbl.Rows.Int64
		}
		if tbl.BytesOnDisk.Valid {
			bytesOnDisk = tbl.BytesOnDisk.Int64
		}
		if tbl.DataUncompressedBytes.Valid {
			dataUncompressed = tbl.DataUncompressedBytes.Int64
		}
		compressionRatio := 0.0
		if bytesOnDisk > 0 {
			compressionRatio = float64(dataUncompressed) / float64(bytesOnDisk)
		}
		tableStatus := response.TableStatus{
			Name:                  tbl.Name,
			Rows:                  rows,
			BytesOnDisk:           bytesOnDisk,
			DataUncompressedBytes: dataUncompressed,
			BytesOnDiskHuman:      formatBytes(bytesOnDisk),
			DataUncompressedHuman: formatBytes(dataUncompressed),
			CompressionRatio:      compressionRatio,
		}
		if _, ok := timestampTables[tbl.Name]; ok {
			minMaxQuery := sq.Select("min(Timestamp)", "max(Timestamp)").
				From(fmt.Sprintf("flamedb.%s", tbl.Name))
			minMax, queryErr := SelectOneSq[struct {
				Min sql.NullTime
				Max sql.NullTime
			}](ctx, c.client, entry, minMaxQuery, func(rows *sql.Rows) (struct {
				Min sql.NullTime
				Max sql.NullTime
			}, error) {
				var out struct {
					Min sql.NullTime
					Max sql.NullTime
				}
				err := rows.Scan(&out.Min, &out.Max)
				return out, err
			})
			if queryErr == nil {
				if minMax.Min.Valid {
					tableStatus.MinTimestamp = &minMax.Min.Time
				}
				if minMax.Max.Valid {
					tableStatus.MaxTimestamp = &minMax.Max.Time
				}
			}
		}
		status.Tables = append(status.Tables, tableStatus)
		status.TotalRows += rows
		status.TotalBytes += bytesOnDisk
	}
	status.TotalBytesHuman = formatBytes(status.TotalBytes)

	return status, nil
}

func formatBytes(value int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	v := float64(value)
	unitIndex := 0
	for v >= 1024 && unitIndex < len(units)-1 {
		v /= 1024
		unitIndex++
	}
	if unitIndex == 0 {
		return fmt.Sprintf("%d %s", value, units[unitIndex])
	}
	return fmt.Sprintf("%.1f %s", v, units[unitIndex])
}
