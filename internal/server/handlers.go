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

package server

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"vihren/internal/flamegraph"
	request "vihren/internal/model/request"
	modelresponse "vihren/internal/model/response"

	"github.com/a8m/rql"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var QueryParser = rql.MustNewParser(rql.Config{
	Model:         request.RQLFilters{},
	FieldSep:      ".",
	LimitMaxValue: 25,
})

var MetricsQueryParser = rql.MustNewParser(rql.Config{
	Model:         request.MetricsRQLFilters{},
	FieldSep:      ".",
	LimitMaxValue: 25,
})

func (s *Server) GetFlamegraph(c *gin.Context) {
	params, query, err := parseParams(request.FlameGraphQuery{}, QueryParser, c)
	if err != nil {
		return
	}

	start := c.GetTime("requestStartTime")
	graph, err := s.DB.GetTopFrames(c.Request.Context(), params, query)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	olapTime := float64(time.Since(start)) / float64(time.Second)
	runtimes := flamegraph.CalcRuntimesDistribution(graph)

	switch params.Format {
	case "flamegraph":
		total, final := graph.BuildFlameGraph()
		result := modelresponse.FlameGraphResponse{
			Name:        "root",
			Value:       total,
			Children:    final,
			OlapTime:    olapTime,
			Percentiles: graph.GetPercentiles(),
		}
		result.SetExecTime(start)
		c.JSON(http.StatusOK, result)

	case "collapsed_file":
		ch := make(chan string)
		go graph.BuildCollapsedFile(ch, runtimes)
		lineNum := 0
		c.Stream(func(w io.Writer) bool {
			line, more := <-ch
			if !more {
				if lineNum == 0 {
					c.Writer.WriteHeader(http.StatusNoContent)
				}
				return false
			}
			_, _ = c.Writer.Write([]byte(line))
			lineNum++
			return true
		})
		for range ch {
		} // drain in case Stream stopped early

	case "svg":
		ch := make(chan string)
		go graph.BuildCollapsedFile(ch, runtimes)
		var collapsed bytes.Buffer
		for line := range ch {
			collapsed.WriteString(line)
		}
		if collapsed.Len() == 0 {
			c.Writer.WriteHeader(http.StatusNoContent)
			return
		}

		exePath, err := os.Executable()
		if err != nil {
			c.String(http.StatusInternalServerError, "cannot locate executable: %v", err)
			return
		}
		scriptPath := filepath.Join(filepath.Dir(exePath), "scripts", "flamegraph.pl")

		cmd := exec.CommandContext(c.Request.Context(), "perl", scriptPath, "--inverted", "--title", params.Service)
		cmd.Stdin = &collapsed
		var svgOut, svgErr bytes.Buffer
		cmd.Stdout = &svgOut
		cmd.Stderr = &svgErr
		if err := cmd.Run(); err != nil {
			log.WithError(err).WithField("stderr", svgErr.String()).Warn("flamegraph.pl failed")
			c.String(http.StatusInternalServerError, "flamegraph.pl failed: %v", err)
			return
		}

		c.Data(http.StatusOK, "image/svg+xml", svgOut.Bytes())

	default:
		c.String(http.StatusBadRequest, "Unknown format")
	}
}

func (s *Server) QueryMeta(c *gin.Context) {
	params, query, err := parseParams(request.QueryMetaQuery{}, QueryParser, c)
	if err != nil {
		return
	}

	mapping := map[string]string{
		"container":     "ContainerName",
		"hostname":      "HostName",
		"instance_type": "InstanceType",
		"pod":           "ContainerEnvName",
	}

	ctx := c.Request.Context()
	var resp modelresponse.ExecTimeInterface

	switch params.LookupTarget {
	case "hostname", "instance_type", "container", "pod":
		field := mapping[params.LookupTarget]
		res, qErr := s.DB.FetchFieldValues(ctx, field, params, query)
		if qErr != nil {
			log.WithError(qErr).Warn("fetch field values failed")
		}
		resp = &modelresponse.Response[[]string]{Result: res}

	case "instance_type_count":
		res, qErr := s.DB.FetchInstanceTypeCount(ctx, params, query)
		if qErr != nil {
			log.WithError(qErr).Warn("fetch instance type count failed")
		}
		resp = &modelresponse.Response[[]modelresponse.InstanceTypeCount]{Result: res}

	case "samples":
		res, qErr := s.DB.FetchSampleCount(ctx, params, query)
		if qErr != nil {
			log.WithError(qErr).Warn("fetch sample count failed")
		}
		resp = &modelresponse.Response[[]modelresponse.SamplePoint]{Result: res}

	default:
		resp = &modelresponse.Response[[]string]{Result: []string{}}
	}

	resp.SetExecTime(c.GetTime("requestStartTime"))
	c.JSON(http.StatusOK, resp)
}

func (s *Server) QueryServices(c *gin.Context) {
	params, _, err := parseParams(request.ServicesQuery{}, nil, c)
	if err != nil {
		return
	}

	res, err := s.DB.FetchServices(c.Request.Context(), params)
	if err != nil {
		log.WithError(err).Warn("fetch services failed")
	}
	resp := modelresponse.Response[[]string]{Result: res}
	resp.SetExecTime(c.GetTime("requestStartTime"))
	c.JSON(http.StatusOK, resp)
}

func (s *Server) QuerySessionsCount(c *gin.Context) {
	params, query, err := parseParams(request.SessionsCountQuery{}, QueryParser, c)
	if err != nil {
		return
	}
	result, err := s.DB.FetchSessionsCount(c.Request.Context(), params, query)
	if err != nil {
		log.WithError(err).Warn("sessions count query failed")
		c.Status(http.StatusNoContent)
		return
	}
	resp := modelresponse.Response[int]{Result: result}
	resp.SetExecTime(c.GetTime("requestStartTime"))
	c.JSON(http.StatusOK, resp)
}

func (s *Server) GetMetricsGraph(c *gin.Context) {
	params, query, err := parseParams(request.MetricsSummaryQuery{}, MetricsQueryParser, c)
	if err != nil {
		return
	}
	fetchResponse, err := s.DB.FetchMetricsGraph(c.Request.Context(), params, query)
	if err != nil {
		log.WithError(err).Warn("metrics graph query failed")
		c.Status(http.StatusNoContent)
		return
	}
	resp := modelresponse.Response[[]modelresponse.MetricsSummary]{Result: fetchResponse}
	resp.SetExecTime(c.GetTime("requestStartTime"))
	c.JSON(http.StatusOK, resp)
}

func (s *Server) GetSummary(c *gin.Context) {
	params, query, err := parseParams(request.SummaryQuery{}, QueryParser, c)
	if err != nil {
		return
	}
	result, err := s.DB.FetchSummaryStats(c.Request.Context(), params, query)
	if err != nil {
		log.WithError(err).Warn("summary query failed")
		c.Status(http.StatusNoContent)
		return
	}
	resp := modelresponse.Response[modelresponse.SummaryStats]{Result: result}
	resp.SetExecTime(c.GetTime("requestStartTime"))
	c.JSON(http.StatusOK, resp)
}

func (s *Server) GetDBStatus(c *gin.Context) {
	start := c.GetTime("requestStartTime")
	status, err := s.DB.FetchDBStatus(c.Request.Context())
	if err != nil {
		log.WithError(err).Warn("db status query failed")
		c.Status(http.StatusNoContent)
		return
	}
	resp := modelresponse.Response[modelresponse.DBStatus]{Result: status}
	resp.SetExecTime(start)
	c.JSON(http.StatusOK, resp)
}
