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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"vihren/internal/config"
)

func newTestQueue(t *testing.T, depth int, sendTimeout time.Duration) *Queue {
	t.Helper()

	dir := t.TempDir()
	q, err := NewQueue(config.QueueConfig{
		PayloadPath: dir,
		QueueDepth:  depth,
		SendTimeout: sendTimeout,
	})
	if err != nil {
		t.Fatalf("NewQueue: %v", err)
	}
	return q
}

func testTask(path, service string) Task {
	return Task{
		Path:      path,
		Service:   service,
		Timestamp: time.Unix(100, 0),
	}
}

func TestPublishConsume_HappyPath_RemovesFile(t *testing.T) {
	q := newTestQueue(t, 10, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got Task
	var gotPayload []byte

	done := make(chan error, 1)
	go func() {
		done <- q.Consume(ctx, func(ctx context.Context, t Task, payload []byte) error {
			got = t
			gotPayload = append([]byte(nil), payload...)
			cancel()
			return nil
		})
	}()

	payload := []byte("hello")
	filename := "a.prof"

	if err := q.Publish(context.Background(), testTask(filename, "svcA"), payload); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	err := <-done
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Consume returned: %v", err)
	}

	if got.Service != "svcA" {
		t.Fatalf("service mismatch: %q", got.Service)
	}
	if string(gotPayload) != "hello" {
		t.Fatalf("payload mismatch: %q", string(gotPayload))
	}
	if _, err := os.Stat(got.Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be removed, stat err=%v", err)
	}
}

func TestConsume_HandlerError_StillRemovesFile(t *testing.T) {
	q := newTestQueue(t, 10, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- q.Consume(ctx, func(ctx context.Context, t Task, payload []byte) error {
			cancel()
			return errors.New("boom")
		})
	}()

	if err := q.Publish(context.Background(), testTask("b.prof", "svcB"), []byte("x")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	err := <-done
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Consume returned: %v", err)
	}

	path := filepath.Join(q.cfg.PayloadPath, "b.prof")
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be removed, stat err=%v", err)
	}
}

func TestPublish_DropOnTimeout_WhenQueueFull(t *testing.T) {
	q := newTestQueue(t, 1, 50*time.Millisecond)

	if err := q.Publish(context.Background(), testTask("1.prof", "svc"), []byte("1")); err != nil {
		t.Fatalf("Publish#1: %v", err)
	}

	err := q.Publish(context.Background(), testTask("2.prof", "svc"), []byte("2"))
	if err == nil {
		t.Fatalf("Publish#2 expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("Publish#2 expected deadline/cancel error, got %v", err)
	}

	path2 := filepath.Join(q.cfg.PayloadPath, "2.prof")
	if _, statErr := os.Stat(path2); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected dropped file to be removed, stat err=%v", statErr)
	}

	path1 := filepath.Join(q.cfg.PayloadPath, "1.prof")
	if _, statErr := os.Stat(path1); statErr != nil {
		t.Fatalf("expected first file to exist, stat err=%v", statErr)
	}
}

func TestPublish_Parallel(t *testing.T) {
	const n = 200
	q := newTestQueue(t, n, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var handled atomic.Int64
	done := make(chan error, 1)

	go func() {
		done <- q.Consume(ctx, func(ctx context.Context, t Task, payload []byte) error {
			handled.Add(1)
			if handled.Load() == n {
				cancel()
			}
			return nil
		})
	}()

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			fn := fmt.Sprintf("%d.prof", i)
			if err := q.Publish(context.Background(), testTask(fn, "svc"), []byte("x")); err != nil {
				t.Errorf("Publish %s: %v", fn, err)
			}
		}()
	}
	wg.Wait()

	err := <-done
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Consume returned: %v", err)
	}

	if handled.Load() != n {
		t.Fatalf("handled=%d want=%d", handled.Load(), n)
	}

	entries, err := os.ReadDir(q.cfg.PayloadPath)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected empty payload dir, left: %v", names)
	}
}
