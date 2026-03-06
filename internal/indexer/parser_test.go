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
	"errors"
	"fmt"
	"strings"
	"testing"
)

func headerJSON(service, version string, appMeta bool, cpu, mem float64) string {
	return fmt.Sprintf(`{"metadata":{"hostname":"host","cloud_info":{"instance_type":"c5"},"run_arguments":{"service_name":"%s","profile_api_version":"%s"}},"metrics":{"cpu_avg":%v,"mem_avg":%v},"application_metadata_enabled":%t}`,
		service, version, cpu, mem, appMeta,
	)
}

func TestParseProfileValidHeader(t *testing.T) {
	p := NewParser(nil)
	input := "#" + headerJSON("svc", "v2", true, 1.2, 3.4) + "\n"
	parsed, err := p.ParseBytes([]byte(input))
	if err != nil {
		t.Fatalf("ParseBytes error: %v", err)
	}
	if parsed.Info.Metadata.Hostname != "host" {
		t.Fatalf("hostname = %q", parsed.Info.Metadata.Hostname)
	}
	if !parsed.WithAppMeta {
		t.Fatalf("WithAppMeta expected true")
	}
	if !parsed.WithContainer {
		t.Fatalf("WithContainer expected true")
	}
	if len(parsed.Samples) != 0 {
		t.Fatalf("unexpected samples: %d", len(parsed.Samples))
	}
}

func TestParseProfileInvalidHeader(t *testing.T) {
	p := NewParser(nil)
	input := "#{" + "\n"
	_, err := p.ParseBytes([]byte(input))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseProfileMissingHeader(t *testing.T) {
	p := NewParser(nil)
	input := "frame1;frame2 5\n"
	_, err := p.ParseBytes([]byte(input))
	if !errors.Is(err, ErrMissingHeader) {
		t.Fatalf("expected ErrMissingHeader, got %v", err)
	}
}

func TestParseSampleLineV1(t *testing.T) {
	p := NewParser(nil)
	input := strings.Join([]string{
		"#" + headerJSON("svc", "v1", false, 0, 0),
		"f1;f2 3",
		"",
	}, "\n")

	parsed, err := p.ParseBytes([]byte(input))
	if err != nil {
		t.Fatalf("ParseBytes error: %v", err)
	}
	if parsed.WithContainer {
		t.Fatalf("WithContainer expected false")
	}
	if len(parsed.Samples) != 1 {
		t.Fatalf("samples len = %d", len(parsed.Samples))
	}
	s := parsed.Samples[0]
	if s.RawContainer != "" {
		t.Fatalf("RawContainer = %q", s.RawContainer)
	}
	if s.Samples != 3 {
		t.Fatalf("Samples = %d", s.Samples)
	}
	if got := strings.Join(s.Stack, ","); got != "f1,f2" {
		t.Fatalf("stack = %q", got)
	}
}

func TestParseSampleLineV2WithContainer(t *testing.T) {
	p := NewParser(nil)
	input := strings.Join([]string{
		"#" + headerJSON("svc", "v2", false, 0, 0),
		"cont;f1;f2 7",
	}, "\n")

	parsed, err := p.ParseBytes([]byte(input))
	if err != nil {
		t.Fatalf("ParseBytes error: %v", err)
	}
	if !parsed.WithContainer {
		t.Fatalf("WithContainer expected true")
	}
	if len(parsed.Samples) != 1 {
		t.Fatalf("samples len = %d", len(parsed.Samples))
	}
	s := parsed.Samples[0]
	if s.RawContainer != "cont" {
		t.Fatalf("RawContainer = %q", s.RawContainer)
	}
	if got := strings.Join(s.Stack, ","); got != "f1,f2" {
		t.Fatalf("stack = %q", got)
	}
}

func TestParseSampleLineV2WithAppMeta(t *testing.T) {
	p := NewParser(nil)
	input := strings.Join([]string{
		"#" + headerJSON("svc", "v2", true, 0, 0),
		"appmeta;cont;f1;f2 4",
	}, "\n")

	parsed, err := p.ParseBytes([]byte(input))
	if err != nil {
		t.Fatalf("ParseBytes error: %v", err)
	}
	if len(parsed.Samples) != 1 {
		t.Fatalf("samples len = %d", len(parsed.Samples))
	}
	s := parsed.Samples[0]
	if s.RawContainer != "cont" {
		t.Fatalf("RawContainer = %q", s.RawContainer)
	}
	if got := strings.Join(s.Stack, ","); got != "f1,f2" {
		t.Fatalf("stack = %q", got)
	}
}

func TestParseSampleLineZeroSamples(t *testing.T) {
	p := NewParser(nil)
	input := strings.Join([]string{
		"#" + headerJSON("svc", "v2", false, 0, 0),
		"cont;f1 0",
		"",
	}, "\n")

	parsed, err := p.ParseBytes([]byte(input))
	if err != nil {
		t.Fatalf("ParseBytes error: %v", err)
	}
	if len(parsed.Samples) != 0 {
		t.Fatalf("unexpected samples: %d", len(parsed.Samples))
	}
}

func BenchmarkParseSampleLine(b *testing.B) {
	p := NewParser(nil)
	line := "appmeta;cont;frame1;frame2;frame3 123"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = p.parseSampleLine(line, true, true)
	}
}
