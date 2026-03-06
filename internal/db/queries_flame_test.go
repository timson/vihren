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

import "testing"

func TestStableParentHash(t *testing.T) {
	parent := stableParentHash(map[int64]int64{
		12: 10,
		4:  10,
		7:  9,
	})
	if parent != 4 {
		t.Fatalf("expected smallest hash among max-sample parents (4), got %d", parent)
	}
}

func TestExpandKnownAncestors(t *testing.T) {
	aggregates := map[int64]*frameAggregate{
		1: {Samples: 40, ParentSamples: map[int64]int64{0: 40}},
		2: {Samples: 60, ParentSamples: map[int64]int64{1: 60}},
		3: {Samples: 55, ParentSamples: map[int64]int64{0: 55}},
	}

	selected := hashSet(topFrameHashes(aggregates, 1))
	if _, ok := selected[2]; !ok {
		t.Fatalf("expected frame 2 to be selected as top frame")
	}

	changed := expandKnownAncestors(selected, aggregates)
	if !changed {
		t.Fatalf("expected ancestor expansion to add known parent")
	}
	if _, ok := selected[1]; !ok {
		t.Fatalf("expected ancestor frame 1 to be added")
	}
	if _, ok := selected[3]; ok {
		t.Fatalf("did not expect unrelated frame 3 to be selected")
	}
}

func TestMissingAncestors(t *testing.T) {
	aggregates := map[int64]*frameAggregate{
		2: {Samples: 60, ParentSamples: map[int64]int64{1: 60}},
	}
	selected := map[int64]struct{}{2: struct{}{}}

	missing := missingAncestors(selected, aggregates)
	if len(missing) != 1 || missing[0] != 1 {
		t.Fatalf("expected missing ancestor [1], got %v", missing)
	}
}
