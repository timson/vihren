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

package indexer

import (
	"context"
	"fmt"

	"vihren/internal/db"

	log "github.com/sirupsen/logrus"
)

type ProfilesWriter struct {
	parser     *Parser
	aggregator *Aggregator
	sink       BlockSink
}

func NewProfilesWriter(out chan<- db.ProfileBlock, normalizer *FrameNormalizer) *ProfilesWriter {
	return NewProfilesWriterWithSink(NewChanSink(out), normalizer)
}

func NewProfilesWriterWithSink(sink BlockSink, normalizer *FrameNormalizer) *ProfilesWriter {
	return &ProfilesWriter{
		parser:     NewParser(normalizer),
		aggregator: NewAggregator(),
		sink:       sink,
	}
}

func (pw *ProfilesWriter) ProcessStackFrameFile(task Task, buf []byte) error {
	return pw.ProcessStackFrameFileCtx(context.Background(), task, buf)
}

func (pw *ProfilesWriter) ProcessStackFrameFileCtx(ctx context.Context, task Task, buf []byte) error {
	entry := log.WithFields(log.Fields{"filename": task.Path, "service": task.Service})
	entry.WithField("bytes", len(buf)).Debug("profile writer start")

	parsed, err := pw.parser.ParseBytes(buf)
	if err != nil {
		return fmt.Errorf("parse stack profile: %w", err)
	}

	block := pw.aggregator.Aggregate(parsed, task)
	if pw.sink != nil && len(block.Stacks) > 0 {
		entry.WithField("stacks", len(block.Stacks)).Debug("profile writer emit")
		if err := pw.sink.WriteProfileBlock(ctx, block); err != nil {
			return fmt.Errorf("emit profile block: %w", err)
		}
	}

	entry.Debug("profile writer done")
	return nil
}
