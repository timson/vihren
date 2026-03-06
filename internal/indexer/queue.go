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
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"time"
	"vihren/internal/config"

	log "github.com/sirupsen/logrus"
)

type Task struct {
	Path      string
	Service   string
	Timestamp time.Time
}

type Queue struct {
	cfg config.QueueConfig
	ch  chan Task
}

func NewQueue(cfg config.QueueConfig) (*Queue, error) {
	_ = os.RemoveAll(cfg.PayloadPath)

	err := os.MkdirAll(cfg.PayloadPath, 0755)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"path":    cfg.PayloadPath,
		"depth":   cfg.QueueDepth,
		"timeout": cfg.SendTimeout,
	}).Info("queue init")
	return &Queue{
		cfg: cfg,
		ch:  make(chan Task, cfg.QueueDepth),
	}, nil
}

func withMinTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok {
		nd := time.Now().Add(d)
		if deadline.Before(nd) {
			return ctx, func() {}
		}
	}
	return context.WithTimeout(ctx, d)
}

func (q *Queue) Publish(ctx context.Context, task Task, data []byte) (err error) {
	path := filepath.Join(q.cfg.PayloadPath, task.Path)
	entry := log.WithFields(log.Fields{"filename": task.Path, "service": task.Service})
	entry.WithField("bytes", len(data)).Debug("queue publish start")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		entry.WithError(err).Error("queue publish open failed")
		return err
	}

	// if return error, let's delete our file
	defer func() {
		if err != nil {
			_ = os.Remove(path)
		}
	}()

	_, err = f.Write(data)
	if err != nil {
		_ = f.Close()
		return err
	}

	if err = f.Close(); err != nil {
		entry.WithError(err).Warn("queue publish close failed")
		return err
	}

	ctxNew, cancel := withMinTimeout(ctx, q.cfg.SendTimeout)
	defer cancel()

	task.Path = path

	select {
	case q.ch <- task:
		entry.Debug("queue publish enqueued")
		return nil
	case <-ctxNew.Done():
		entry.WithError(ctxNew.Err()).Warn("queue publish timeout")
		_ = os.Remove(path)
		return ctxNew.Err()
	}
}

func readPayload(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	if filepath.Ext(path) != ".gz" {
		return io.ReadAll(f)
	}

	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()

	return io.ReadAll(zr)
}

func (q *Queue) Consume(ctx context.Context, handler func(ctx context.Context, t Task, payload []byte) error) error {
	for {
		select {
		case <-ctx.Done():
			log.WithError(ctx.Err()).Info("queue consume ctx done")
			return ctx.Err()
		case t, ok := <-q.ch:
			if !ok {
				log.Info("queue consume channel closed")
				return nil
			}
			func() {
				defer func() { _ = os.Remove(t.Path) }()

				data, err := readPayload(t.Path)
				entry := log.WithFields(log.Fields{"filename": t.Path, "service": t.Service})
				if err != nil {
					entry.WithError(err).Error("queue consume read failed")
					return
				}

				entry.WithField("bytes", len(data)).Debug("queue consume start")
				if err := handler(ctx, t, data); err != nil {
					entry.WithError(err).Error("queue consume handler error")
					return
				}
				entry.Debug("queue consume done")
			}()

		}
	}
}

func (q *Queue) Close() {
	close(q.ch)
}
