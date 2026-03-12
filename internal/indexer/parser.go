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
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

)

var ErrMissingHeader = errors.New("missing header")

type Sample struct {
	RawContainer string
	Stack        []string
	Samples      int64
}

type ParsedProfile struct {
	Info          StackFileInfo
	WithAppMeta   bool
	WithContainer bool
	Samples       []Sample
}

type StackFileInfo struct {
	Metadata struct {
		Hostname  string `json:"hostname"`
		CloudInfo struct {
			InstanceType string `json:"instance_type"`
		} `json:"cloud_info"`
		RunArguments struct {
			ServiceName       string `json:"service_name"`
			ProfileAPIVersion string `json:"profile_api_version"`
		} `json:"run_arguments"`
	} `json:"metadata"`

	Metrics struct {
		CPUAvg    float64 `json:"cpu_avg"`
		MemoryAvg float64 `json:"mem_avg"`
	} `json:"metrics"`

	ApplicationMetadataEnabled bool `json:"application_metadata_enabled"`
}

type Parser struct {
	normalizer *FrameNormalizer
}

func NewParser(normalizer *FrameNormalizer) *Parser {
	return &Parser{normalizer: normalizer}
}

func (p *Parser) ParseBytes(buf []byte) (ParsedProfile, error) {
	return p.ParseReader(bytes.NewReader(buf))
}

func (p *Parser) ParseReader(r io.Reader) (ParsedProfile, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, ScannerBufSize), MaxScannerBufSize)

	var (
		info          StackFileInfo
		withAppMeta   bool
		withContainer bool
		headerSeen    bool
		samples       []Sample
	)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			parsed, enabled, err := parseHeaderLine(line)
			if err != nil {
				return ParsedProfile{}, err
			}
			info = parsed
			withAppMeta = enabled
			withContainer = info.Metadata.RunArguments.ProfileAPIVersion != V1Prefix
			headerSeen = true
			continue
		}
		if !headerSeen {
			return ParsedProfile{}, ErrMissingHeader
		}

		sample, ok, err := p.parseSampleLine(line, withContainer, withAppMeta)
		if err != nil {
			return ParsedProfile{}, err
		}
		if !ok {
			continue
		}
		samples = append(samples, sample)
	}
	if err := sc.Err(); err != nil {
		return ParsedProfile{}, err
	}

	return ParsedProfile{
		Info:          info,
		WithAppMeta:   withAppMeta,
		WithContainer: withContainer,
		Samples:       samples,
	}, nil
}

func parseHeaderLine(line string) (info StackFileInfo, withAppMeta bool, err error) {
	if len(line) == 0 || line[0] != '#' {
		return StackFileInfo{}, false, fmt.Errorf("not a header line")
	}
	if err := json.Unmarshal([]byte(line[1:]), &info); err != nil {
		return StackFileInfo{}, false, err
	}
	return info, info.ApplicationMetadataEnabled, nil
}

func (p *Parser) parseSampleLine(line string, withContainer bool, withAppMeta bool) (Sample, bool, error) {
	lastSpace := strings.LastIndexByte(line, ' ')
	if lastSpace < 0 {
		return Sample{}, false, fmt.Errorf("invalid sample line: %q", line)
	}

	samplesStr := strings.TrimSpace(line[lastSpace+1:])
	if samplesStr == "" {
		return Sample{}, false, fmt.Errorf("invalid samples count in %q", line)
	}
	callstackPart := strings.TrimSpace(line[:lastSpace])
	if callstackPart == "" {
		return Sample{}, false, fmt.Errorf("invalid sample line: %q", line)
	}

	samples, err := strconv.ParseInt(samplesStr, 10, 64)
	if err != nil {
		return Sample{}, false, fmt.Errorf("invalid samples count in %q: %w", line, err)
	}
	if samples == 0 {
		return Sample{}, false, nil
	}

	rawContainer, stackPart, err := splitCallstack(callstackPart, withContainer, withAppMeta)
	if err != nil {
		return Sample{}, false, err
	}

	if stackPart == "" {
		return Sample{}, false, nil
	}

	stackPart = p.normalizeCallstack(stackPart)

	stack := make([]string, 0, 1+strings.Count(stackPart, ";"))
	for _, frame := range strings.Split(stackPart, ";") {
		frame = strings.TrimSpace(frame)
		if frame == "" {
			continue
		}
		stack = append(stack, frame)
	}
	if len(stack) == 0 {
		return Sample{}, false, nil
	}

	return Sample{RawContainer: rawContainer, Stack: stack, Samples: samples}, true, nil
}

func (p *Parser) normalizeCallstack(callstack string) string {
	if p.normalizer == nil {
		return callstack
	}
	if !p.normalizer.ShouldNormalize(callstack) {
		return callstack
	}
	return p.normalizer.Normalize(callstack)
}

func splitCallstack(callstackPart string, withContainer bool, withAppMeta bool) (string, string, error) {
	if !withContainer {
		return "", strings.TrimSpace(callstackPart), nil
	}

	firstSep := strings.IndexByte(callstackPart, ';')
	if firstSep < 0 {
		return "", "", fmt.Errorf("invalid sample line (missing container): %q", callstackPart)
	}

	if withAppMeta {
		secondSep := strings.IndexByte(callstackPart[firstSep+1:], ';')
		if secondSep < 0 {
			return "", "", fmt.Errorf("invalid sample line (missing meta/container): %q", callstackPart)
		}
		secondSep += firstSep + 1
		raw := strings.TrimSpace(callstackPart[firstSep+1 : secondSep])
		stack := strings.TrimSpace(callstackPart[secondSep+1:])
		return raw, stack, nil
	}

	raw := strings.TrimSpace(callstackPart[:firstSep])
	stack := strings.TrimSpace(callstackPart[firstSep+1:])
	return raw, stack, nil
}
