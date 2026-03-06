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
	"fmt"
	"sort"
	"time"
)

const (
	ResRaw   string = "raw"
	ResHour  string = "hour"
	ResDay   string = "day"
	ResMulti string = "multi"
)

const (
	TableRaw string = "raw"
	Table1H  string = "1hour"
	Table1D  string = "1day"
)

type Chunk struct {
	Table string
	Start time.Time // inclusive
	End   time.Time // exclusive
}

func BuildChunks(start, end time.Time, res string) ([]Chunk, error) {
	if !start.Before(end) {
		return nil, fmt.Errorf("invalid range: start=%s end=%s", start, end)
	}

	switch res {
	case ResRaw:
		return []Chunk{{Table: TableRaw, Start: start, End: end}}, nil
	case ResHour:
		return []Chunk{{Table: Table1H, Start: start, End: end}}, nil
	case ResDay:
		return []Chunk{{Table: Table1D, Start: start, End: end}}, nil
	case ResMulti:
		chunks := splitMulti(start, end)
		return mergeChunks(chunks), nil
	default:
		return nil, fmt.Errorf("unknown string: %q", res)
	}
}

func splitMulti(start, end time.Time) []Chunk {
	var out []Chunk

	if !isHourAligned(start) {
		leftEnd := minTime(end, nextHour(start))
		out = append(out, Chunk{Table: TableRaw, Start: start, End: leftEnd})
		start = leftEnd
		if !start.Before(end) {
			return out
		}
	}

	if !isHourAligned(end) {
		rightStart := maxTime(start, startOfHour(end))
		out = append(out, Chunk{Table: TableRaw, Start: rightStart, End: end})
		end = rightStart
		if !start.Before(end) {
			return out
		}
	}

	firstDayStart := startOfDay(start)
	if start.After(firstDayStart) {
		firstDayStart = nextDay(start)
	}
	lastDayStart := startOfDay(end)

	if firstDayStart.Before(lastDayStart) {
		if start.Before(firstDayStart) {
			out = append(out, Chunk{Table: Table1H, Start: start, End: firstDayStart})
		}
		out = append(out, Chunk{Table: Table1D, Start: firstDayStart, End: lastDayStart})
		if lastDayStart.Before(end) {
			out = append(out, Chunk{Table: Table1H, Start: lastDayStart, End: end})
		}
		return out
	}

	out = append(out, Chunk{Table: Table1H, Start: start, End: end})
	return out
}

func mergeChunks(ch []Chunk) []Chunk {
	if len(ch) == 0 {
		return ch
	}

	sort.Slice(ch, func(i, j int) bool {
		if !ch[i].Start.Equal(ch[j].Start) {
			return ch[i].Start.Before(ch[j].Start)
		}
		if !ch[i].End.Equal(ch[j].End) {
			return ch[i].End.Before(ch[j].End)
		}
		return ch[i].Table < ch[j].Table
	})

	out := make([]Chunk, 0, len(ch))
	cur := ch[0]

	for i := 1; i < len(ch); i++ {
		n := ch[i]

		if n.Table == cur.Table && !n.Start.After(cur.End) { // n.Start <= cur.End
			if n.End.After(cur.End) {
				cur.End = n.End
			}
			continue
		}

		if cur.Start.Before(cur.End) {
			out = append(out, cur)
		}
		cur = n
	}

	if cur.Start.Before(cur.End) {
		out = append(out, cur)
	}
	return out
}

func startOfHour(t time.Time) time.Time { return t.Truncate(time.Hour) }
func nextHour(t time.Time) time.Time    { return startOfHour(t).Add(time.Hour) }
func isHourAligned(t time.Time) bool    { return t.Equal(startOfHour(t)) }

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
func nextDay(t time.Time) time.Time { return startOfDay(t).Add(24 * time.Hour) }

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
