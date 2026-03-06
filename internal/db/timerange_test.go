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
	"testing"
	"time"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse time %q: %v", s, err)
	}
	return tt.UTC()
}

func assertChunks(t *testing.T, got []Chunk, want []Chunk) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got=%d want=%d\ngot=%v\nwant=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i].Table != want[i].Table {
			t.Fatalf("[%d] table mismatch: got=%q want=%q", i, got[i].Table, want[i].Table)
		}
		if !got[i].Start.Equal(want[i].Start) {
			t.Fatalf("[%d] start mismatch: got=%s want=%s", i, got[i].Start.Format(time.RFC3339), want[i].Start.Format(time.RFC3339))
		}
		if !got[i].End.Equal(want[i].End) {
			t.Fatalf("[%d] end mismatch: got=%s want=%s", i, got[i].End.Format(time.RFC3339), want[i].End.Format(time.RFC3339))
		}
		if !got[i].Start.Before(got[i].End) {
			t.Fatalf("[%d] invalid chunk: start !< end (%s !< %s)", i, got[i].Start, got[i].End)
		}
	}
}

func TestBuildChunks_InvalidRange(t *testing.T) {
	start := mustTime(t, "2026-01-01T10:00:00Z")
	end := mustTime(t, "2026-01-01T10:00:00Z")

	_, err := BuildChunks(start, end, ResRaw)
	if err == nil {
		t.Fatalf("expected error for start==end")
	}
}

func TestBuildChunks_UnknownResolution(t *testing.T) {
	start := mustTime(t, "2026-01-01T10:00:00Z")
	end := mustTime(t, "2026-01-01T11:00:00Z")

	_, err := BuildChunks(start, end, "nope")
	if err == nil {
		t.Fatalf("expected error for unknown res")
	}
}

func TestBuildChunks_SingleTable(t *testing.T) {
	start := mustTime(t, "2026-01-01T10:00:00Z")
	end := mustTime(t, "2026-01-01T11:00:00Z")

	got, err := BuildChunks(start, end, ResHour)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	want := []Chunk{{Table: Table1H, Start: start, End: end}}
	assertChunks(t, got, want)
}

func TestBuildChunks_Multi_LessThanHourButUnaligned(t *testing.T) {
	// 10:12 -> 10:40, оба не на границе часа
	start := mustTime(t, "2026-01-01T10:12:00Z")
	end := mustTime(t, "2026-01-01T10:40:00Z")

	got, err := BuildChunks(start, end, ResMulti)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// splitMulti: левый raw [start, end) + правый raw может попытаться добавиться,
	// но mergeChunks склеит в один raw.
	want := []Chunk{{Table: TableRaw, Start: start, End: end}}
	assertChunks(t, got, want)
}

func TestBuildChunks_Multi_SpansHoursNoFullDays(t *testing.T) {
	// 10:12 -> 14:40
	start := mustTime(t, "2026-01-01T10:12:00Z")
	end := mustTime(t, "2026-01-01T14:40:00Z")

	got, err := BuildChunks(start, end, ResMulti)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// raw: [10:12,11:00) и [14:00,14:40)
	// hour: [11:00,14:00)
	want := []Chunk{
		{Table: TableRaw, Start: start, End: mustTime(t, "2026-01-01T11:00:00Z")},
		{Table: Table1H, Start: mustTime(t, "2026-01-01T11:00:00Z"), End: mustTime(t, "2026-01-01T14:00:00Z")},
		{Table: TableRaw, Start: mustTime(t, "2026-01-01T14:00:00Z"), End: end},
	}
	assertChunks(t, got, want)
}

func TestBuildChunks_Multi_WithFullDaysInside(t *testing.T) {
	// 2026-01-01 10:12 -> 2026-01-05 14:40
	start := mustTime(t, "2026-01-01T10:12:00Z")
	end := mustTime(t, "2026-01-05T14:40:00Z")

	got, err := BuildChunks(start, end, ResMulti)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// raw: [start, 2026-01-01 11:00) и [2026-01-05 14:00, end)
	// hour: [2026-01-01 11:00, 2026-01-02 00:00) и [2026-01-05 00:00, 2026-01-05 14:00)
	// day:  [2026-01-02 00:00, 2026-01-05 00:00)
	want := []Chunk{
		{Table: TableRaw, Start: start, End: mustTime(t, "2026-01-01T11:00:00Z")},
		{Table: Table1H, Start: mustTime(t, "2026-01-01T11:00:00Z"), End: mustTime(t, "2026-01-02T00:00:00Z")},
		{Table: Table1D, Start: mustTime(t, "2026-01-02T00:00:00Z"), End: mustTime(t, "2026-01-05T00:00:00Z")},
		{Table: Table1H, Start: mustTime(t, "2026-01-05T00:00:00Z"), End: mustTime(t, "2026-01-05T14:00:00Z")},
		{Table: TableRaw, Start: mustTime(t, "2026-01-05T14:00:00Z"), End: end},
	}
	assertChunks(t, got, want)
}

func TestBuildChunks_Multi_ExactlyAlignedHours(t *testing.T) {
	// ровно по границам часа => raw нет, должны быть только 1hour (и возможно 1day если много)
	start := mustTime(t, "2026-01-01T10:00:00Z")
	end := mustTime(t, "2026-01-01T14:00:00Z")

	got, err := BuildChunks(start, end, ResMulti)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	want := []Chunk{
		{Table: Table1H, Start: start, End: end},
	}
	assertChunks(t, got, want)
}
