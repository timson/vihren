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
	"vihren/internal/config"
	"vihren/internal/db"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type Indexer struct {
	workers errgroup.Group
	Queue   *Queue
	PWriter *ProfilesWriter
	nWorker int
}

func NewIndexer(cfg config.IndexerConfig, queue *Queue, out chan<- db.ProfileBlock) (*Indexer, error) {
	n, err := NewFrameNormalizer(cfg.NormalizerPath)
	if err != nil {
		return nil, err
	}

	indexer := Indexer{
		Queue:   queue,
		PWriter: NewProfilesWriter(out, n),
		nWorker: cfg.Workers,
	}
	log.WithFields(log.Fields{
		"workers":    cfg.Workers,
		"normalizer": cfg.NormalizerPath,
	}).Info("indexer init")
	return &indexer, nil
}

func (i *Indexer) Start() {
	for w := 0; w < i.nWorker; w++ {
		log.WithField("worker_id", w).Debug("indexer worker start")
		i.workers.Go(func() error {
			return Worker(context.Background(), i.Queue, i.PWriter)
		})
	}
}
