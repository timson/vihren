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

	"vihren/internal/db"
)

type BlockSink interface {
	WriteProfileBlock(ctx context.Context, b db.ProfileBlock) error
}

type ChanSink struct {
	ch chan<- db.ProfileBlock
}

func NewChanSink(ch chan<- db.ProfileBlock) *ChanSink {
	return &ChanSink{ch: ch}
}

func (s *ChanSink) WriteProfileBlock(ctx context.Context, b db.ProfileBlock) error {
	if s == nil || s.ch == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.ch <- b:
		return nil
	}
}
