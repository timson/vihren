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

package flamegraph

import (
	"testing"
	"vihren/internal/stackframe"
)

func TestPrepareFrames_ClampsChildrenTopologically(t *testing.T) {
	graph := Graph{
		Frames: map[int64]stackframe.Frame{
			1: {Hash: 1, ParentHash: 0, Samples: 50},
			2: {Hash: 2, ParentHash: 1, Samples: 80},
			3: {Hash: 3, ParentHash: 2, Samples: 70},
		},
	}

	glitches, err := graph.PrepareFrames(0)
	if err != nil {
		t.Fatalf("PrepareFrames returned error: %v", err)
	}
	if glitches != 2 {
		t.Fatalf("expected 2 glitches, got %d", glitches)
	}
	if graph.Frames[2].Samples != 50 {
		t.Fatalf("expected child to be clamped to 50, got %d", graph.Frames[2].Samples)
	}
	if graph.Frames[3].Samples != 50 {
		t.Fatalf("expected grandchild to be clamped to 50, got %d", graph.Frames[3].Samples)
	}
	if !graph.Frames[1].Children[2] {
		t.Fatalf("expected edge 1->2 to exist")
	}
	if !graph.Frames[2].Children[3] {
		t.Fatalf("expected edge 2->3 to exist")
	}
}

func TestPrepareFrames_LimitRetainsAncestors(t *testing.T) {
	graph := Graph{
		Frames: map[int64]stackframe.Frame{
			1: {Hash: 1, ParentHash: 0, Samples: 30},
			2: {Hash: 2, ParentHash: 1, Samples: 60},
			3: {Hash: 3, ParentHash: 0, Samples: 55},
		},
	}

	_, err := graph.PrepareFrames(1)
	if err != nil {
		t.Fatalf("PrepareFrames returned error: %v", err)
	}

	if len(graph.Frames) != 2 {
		t.Fatalf("expected 2 frames after limiting with ancestor closure, got %d", len(graph.Frames))
	}
	if _, ok := graph.Frames[1]; !ok {
		t.Fatalf("expected ancestor frame 1 to be kept")
	}
	if _, ok := graph.Frames[2]; !ok {
		t.Fatalf("expected top frame 2 to be kept")
	}
	if _, ok := graph.Frames[3]; ok {
		t.Fatalf("did not expect unrelated frame 3 to be kept")
	}
	if graph.Frames[2].Samples != 30 {
		t.Fatalf("expected child frame to be clamped to ancestor sample (30), got %d", graph.Frames[2].Samples)
	}
}
