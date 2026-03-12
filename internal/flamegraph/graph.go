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
	"fmt"
	"sort"
	"strconv"
	"vihren/internal/stackframe"

	"github.com/ianlancetaylor/demangle"
	"github.com/montanaflynn/stats"
	log "github.com/sirupsen/logrus"
)

type Graph struct {
	Frames         map[int64]stackframe.Frame
	percentiles    map[string]string
	rootFrames     []int64
	EnrichWithLang bool
}

type ResponseFrame struct {
	Name        string          `json:"name"`
	Suffix      string          `json:"suffix,omitempty"`
	Value       int64           `json:"value"`
	Children    []ResponseFrame `json:"children"`
	Language    string          `json:"language,omitempty"`
	SpecialType string          `json:"specialType,omitempty"`
}

func NewGraph() Graph {
	return Graph{
		Frames:         make(map[int64]stackframe.Frame),
		EnrichWithLang: true,
	}
}

func (graph *Graph) PrepareFrames(limitFrames int) (int, error) {
	graph.rootFrames = make([]int64, 0)
	graph.percentiles = make(map[string]string)
	if len(graph.Frames) == 0 {
		return 0, nil
	}

	if limitFrames > 0 && limitFrames < len(graph.Frames) {
		hashes := make([]int64, 0, len(graph.Frames))
		for hash := range graph.Frames {
			hashes = append(hashes, hash)
		}
		sort.Slice(hashes, func(i, j int) bool {
			si := graph.Frames[hashes[i]].Samples
			sj := graph.Frames[hashes[j]].Samples
			if si == sj {
				return hashes[i] < hashes[j]
			}
			return si > sj
		})
		selected := make(map[int64]struct{}, limitFrames)
		for _, hash := range hashes[:limitFrames] {
			selected[hash] = struct{}{}
		}

		for {
			added := false
			for hash := range selected {
				parentHash := graph.Frames[hash].ParentHash
				if parentHash == 0 {
					continue
				}
				if _, ok := graph.Frames[parentHash]; !ok {
					continue
				}
				if _, ok := selected[parentHash]; ok {
					continue
				}
				selected[parentHash] = struct{}{}
				added = true
			}
			if !added {
				break
			}
		}

		filtered := make(map[int64]stackframe.Frame, len(selected))
		for hash := range selected {
			filtered[hash] = graph.Frames[hash]
		}
		graph.Frames = filtered
	}

	log.WithField("frame_count", len(graph.Frames)).Debug("fetched frames")

	for hash, frame := range graph.Frames {
		frame.Children = make(map[int64]bool)
		frame.IsRoot = false
		graph.Frames[hash] = frame
	}

	rootSet := make(map[int64]struct{})
	for hash, frame := range graph.Frames {
		if frame.ParentHash == 0 {
			frame.IsRoot = true
			graph.Frames[hash] = frame
			rootSet[hash] = struct{}{}
			continue
		}

		parent, ok := graph.Frames[frame.ParentHash]
		if !ok {
			frame.IsRoot = true
			graph.Frames[hash] = frame
			rootSet[hash] = struct{}{}
			log.WithFields(log.Fields{
				"parent_hash": frame.ParentHash,
				"frame":       frame.Name,
				"frame_hash":  frame.Hash,
			}).Warn("parent frame not found")
			continue
		}
		parent.Children[hash] = true
		graph.Frames[frame.ParentHash] = parent
	}

	// BFS: enforce parent ≤ child samples invariant.
	glitchesFound := 0
	queue := make([]int64, 0, len(rootSet))
	for hash := range rootSet {
		queue = append(queue, hash)
	}
	visited := make(map[int64]bool, len(graph.Frames))
	for i := 0; i < len(queue); i++ {
		hash := queue[i]
		if visited[hash] {
			continue
		}
		visited[hash] = true

		parentSamples := graph.Frames[hash].Samples
		for childHash := range graph.Frames[hash].Children {
			child := graph.Frames[childHash]
			if child.Samples > parentSamples {
				child.Samples = parentSamples
				graph.Frames[childHash] = child
				glitchesFound++
			}
			queue = append(queue, childHash)
		}
	}
	for hash := range graph.Frames {
		if !visited[hash] {
			rootSet[hash] = struct{}{}
		}
	}
	graph.rootFrames = make([]int64, 0, len(rootSet))
	for hash := range rootSet {
		graph.rootFrames = append(graph.rootFrames, hash)
	}

	for hash := range graph.Frames {
		frame := graph.Frames[hash]
		lang, specialType := stackframe.IdentFrameLangAndSpecialType(frame.Name)
		frame.Lang = lang
		frame.SpecialType = specialType
		if lang == stackframe.Rust || lang == stackframe.Cpp {
			if demangled, err := demangle.ToString(frame.Name); err == nil {
				frame.Name = demangled
			}
		}
		graph.Frames[hash] = frame
	}

	samples := make([]float64, 0, len(graph.Frames))
	for _, v := range graph.Frames {
		samples = append(samples, float64(v.Samples))
	}

	if len(samples) > 0 {
		for percent := 1; percent <= 100; percent++ {
			pVal, _ := stats.Percentile(samples, float64(percent))
			graph.percentiles[strconv.Itoa(percent)] = strconv.FormatInt(int64(pVal), 10)
		}
	}
	return glitchesFound, nil
}

func (graph *Graph) UpdateFrames(frames map[int64]stackframe.Frame) {
	for hash, frame := range frames {
		if _, ok := graph.Frames[hash]; !ok {
			graph.Frames[hash] = frame
		} else {
			frame.Samples += graph.Frames[hash].Samples
			graph.Frames[hash] = frame
		}
	}
}

func (graph *Graph) GetPercentiles() map[string]string {
	return graph.percentiles
}

func (graph *Graph) BuildFlameGraph() (int64, []ResponseFrame) {
	var generateChildren func(hash int64, parent int64, recursionLevel int64) ResponseFrame
	result := make([]ResponseFrame, 0)
	var total int64
	var recursionLevel int64

	generateChildren = func(hash int64, parent int64, recursionLevel int64) ResponseFrame {
		frame := graph.Frames[hash]
		childs := make([]int64, 0)
		for childHash := range frame.Children {
			childs = append(childs, childHash)
		}

		if frame.Lang == stackframe.MayBeGo {
			parentFrame := graph.Frames[parent]
			if parentFrame.Lang == stackframe.Go {
				frame.Lang = stackframe.Go
			} else {
				frame.Lang = stackframe.Other
			}
			graph.Frames[hash] = frame
		}
		newChildren := make([]ResponseFrame, 0, len(childs))
		for _, child := range childs {
			newChildren = append(newChildren, generateChildren(child, hash, recursionLevel+1))
		}
		sort.Slice(newChildren, func(i, j int) bool {
			return newChildren[i].Name < newChildren[j].Name
		})

		name, suffix := frame.GetTruncatedNameAndSuffix()
		return ResponseFrame{Name: name, Suffix: suffix, Value: graph.Frames[hash].Samples,
			Children: newChildren, Language: frame.Lang, SpecialType: frame.SpecialType}
	}

	sort.Slice(graph.rootFrames, func(i, j int) bool {
		return graph.Frames[graph.rootFrames[i]].Name < graph.Frames[graph.rootFrames[j]].Name
	})
	for _, frame := range graph.rootFrames {
		res := generateChildren(frame, 0, recursionLevel)
		result = append(result, res)
		total += graph.Frames[frame].Samples
	}

	return total, result
}

func (graph *Graph) BuildCollapsedFile(out chan string, runtimes map[string]float64) {
	var generateChildren func(seq []int64, path string, rootHash int64) int64
	defer close(out)

	generateChildren = func(seq []int64, path string, rootHash int64) int64 {
		var sumWeight int64
		for _, hash := range seq {
			childs := make([]int64, 0)
			for childHash := range graph.Frames[hash].Children {
				childs = append(childs, childHash)
			}
			frame := graph.Frames[hash]
			var weight int64

			if path == "" {
				rootHash = hash
			}

			if len(childs) > 0 {
				if path != "" {
					weight = frame.Samples - generateChildren(childs, fmt.Sprintf("%s;%s", path, frame.Name), rootHash)
				} else {
					weight = frame.Samples - generateChildren(childs, frame.Name, rootHash)
				}
			} else {
				weight = frame.Samples
			}
			if weight > 0 {
				if path == "" {
					out <- fmt.Sprintf("%s %d\n", frame.Name, weight)
				} else {
					out <- fmt.Sprintf("%s;%s %d\n", path, frame.Name, weight)
				}
			}
			sumWeight += frame.Samples
		}
		return sumWeight
	}

	if len(graph.rootFrames) == 0 {
		return
	}
	generateChildren(graph.rootFrames, "", 0)
}

func sortChildrenBySamples(graph *Graph, frameId int64) []int64 {
	frames := make([]stackframe.Frame, 0)
	for hash := range graph.Frames[frameId].Children {
		frames = append(frames, graph.Frames[hash])
	}

	sort.Slice(frames, func(i, j int) bool {
		return frames[i].Samples > frames[j].Samples
	})

	childrenSamplesSorted := make([]int64, 0, len(frames))
	for _, frame := range frames {
		childrenSamplesSorted = append(childrenSamplesSorted, frame.Hash)
	}
	return childrenSamplesSorted
}

func findRuntimes(graph *Graph, frameId int64, recursionLevel int64, frames map[int64]stackframe.Frame) {
	sortedChildren := sortChildrenBySamples(graph, frameId)
	for _, childId := range sortedChildren {
		frame := graph.Frames[childId]
		if frame.Lang != "" && frame.Lang != stackframe.Other && frame.Lang != stackframe.MayBeGo && frame.Lang != stackframe.Kernel {
			frames[frame.Hash] = frame
			return
		}
		if len(frame.Children) == 0 || recursionLevel > 100 {
			continue
		}
		findRuntimes(graph, frame.Hash, recursionLevel+1, frames)
	}
}

func determineRuntime(graph *Graph, frameHash int64) string {
	framesMap := make(map[int64]stackframe.Frame)
	findRuntimes(graph, frameHash, 1, framesMap)

	frames := make([]stackframe.Frame, 0, len(framesMap))
	for _, v := range framesMap {
		frames = append(frames, v)
	}
	if len(frames) == 0 {
		return stackframe.Other
	}
	sort.Slice(frames, func(i, j int) bool {
		return frames[i].Samples > frames[j].Samples
	})
	return frames[0].Lang
}

func CalcRuntimesDistribution(graph *Graph) map[string]float64 {
	var allSamples int64
	runtimes := make(map[string]int64)

	for _, rootFrameHash := range graph.rootFrames {
		rootFrame := graph.Frames[rootFrameHash]
		runtime := determineRuntime(graph, rootFrameHash)
		runtimes[runtime] += rootFrame.Samples
		allSamples += rootFrame.Samples
	}

	runtimesPercentage := make(map[string]float64, len(runtimes))
	for runtime, samples := range runtimes {
		runtimesPercentage[runtime] = float64(samples) / float64(allSamples)
	}
	return runtimesPercentage
}
