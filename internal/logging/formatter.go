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

package logging

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type KVFormatter struct {
	TimestampFormat string
	UseColors       bool
	SortKeys        bool
}

func (f *KVFormatter) Format(e *logrus.Entry) ([]byte, error) {
	var b bytes.Buffer

	tsFmt := f.TimestampFormat
	if tsFmt == "" {
		tsFmt = "2006-01-02 15:04:05"
	}

	// timestamp
	b.WriteString(e.Time.Format(tsFmt))
	b.WriteByte(' ')

	// level
	level := strings.ToUpper(e.Level.String())
	if f.UseColors {
		level = colorizeLevel(e.Level, level)
	}
	b.WriteString(level)
	b.WriteByte(' ')

	// message
	b.WriteString(e.Message)

	// fields
	if len(e.Data) > 0 {
		keys := make([]string, 0, len(e.Data))
		for k := range e.Data {
			keys = append(keys, k)
		}
		if f.SortKeys {
			sort.Strings(keys)
		}

		for _, k := range keys {
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(formatValue(e.Data[k]))
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func formatValue(v any) string {
	switch x := v.(type) {
	case string:
		// quote only when needed (spaces, tabs, quotes, '=' etc.)
		if x == "" || strings.ContainsAny(x, " \t\n\r\"=") {
			return strconv.Quote(x)
		}
		return x
	case time.Duration:
		return x.String()
	default:
		return fmt.Sprint(v)
	}
}

func colorizeLevel(lvl logrus.Level, s string) string {
	// ANSI colors (simple)
	const (
		red    = "\x1b[31m"
		yellow = "\x1b[33m"
		green  = "\x1b[32m"
		blue   = "\x1b[34m"
		gray   = "\x1b[90m"
		reset  = "\x1b[0m"
	)

	color := reset
	switch lvl {
	case logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel:
		color = red
	case logrus.WarnLevel:
		color = yellow
	case logrus.InfoLevel:
		color = green
	case logrus.DebugLevel:
		color = blue
	case logrus.TraceLevel:
		color = gray
	}
	return color + s + reset
}
