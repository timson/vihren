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
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mathrand "math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vihren/internal/indexer"
	"vihren/internal/model/request"
	modelresponse "vihren/internal/model/response"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (s *Server) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) Logs(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"status": "ok"})
}

func (s *Server) NewProfileV2(c *gin.Context) {
	apiKey := c.GetHeader("Gprofiler-Api-Key")
	serviceNameHeader := c.GetHeader("Gprofiler-Service-Name")
	if apiKey == "" || serviceNameHeader == "" {
		log.Warn("missing gprofiler headers")
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing gprofiler headers"})
		return
	}

	if c.GetHeader("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(c.Request.Body)
		if err != nil {
			log.WithError(err).Warn("failed to create gzip reader")
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid gzip body"})
			return
		}
		defer func() { _ = gr.Close() }()

		decoded, err := io.ReadAll(gr)
		if err != nil {
			log.WithError(err).Warn("failed to decode gzip body")
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid gzip body"})
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(decoded))
	}

	var agentData request.AgentData
	if err := c.ShouldBindJSON(&agentData); err != nil {
		log.WithError(err).Warn("failed to parse json body")
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid json body"})
		return
	}

	profile := agentData.Profile
	firstLine := profile
	if idx := strings.Index(profile, "\n"); idx != -1 {
		firstLine = profile[:idx]
	}
	if firstLine == "" || firstLine[0] != '#' {
		log.WithField("header", firstLine).Warn("invalid profile header")
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid profile header"})
		return
	}

	var profileHeader map[string]any
	if err := json.Unmarshal([]byte(firstLine[1:]), &profileHeader); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid profile metadata"})
		return
	}

	gpid, err := parseGPID(agentData.GPID)
	if err != nil {
		log.WithError(err).Warn("invalid gpid")
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid gpid"})
		return
	}

	metadataRaw, ok := profileHeader["metadata"].(map[string]any)
	if !ok {
		log.Warn("agent metadata missing")
		c.JSON(http.StatusBadRequest, gin.H{"message": "agent metadata missing"})
		return
	}

	hostname, _ := metadataRaw["hostname"].(string)
	if hostname == "" {
		log.Warn("hostname missing from metadata")
		c.JSON(http.StatusBadRequest, gin.H{"message": "hostname missing from metadata"})
		return
	}

	metadataRaw["public_ip"] = clientIP(c.Request)

	if cloudInfo, ok := metadataRaw["cloud_info"].(map[string]any); ok {
		if instanceType, _ := cloudInfo["instance_type"].(string); instanceType != "" {
			metadataRaw["instance_type"] = instanceType
		}
	}
	if _, ok := metadataRaw["instance_type"]; !ok {
		metadataRaw["instance_type"] = ""
	}

	agentVersion, _ := metadataRaw["agent_version"].(string)
	if agentVersion == "" {
		log.Warn("agent_version missing from metadata")
		c.JSON(http.StatusBadRequest, gin.H{"message": "agent_version missing from metadata"})
		return
	}

	profileFileName := getProfileFileName(agentData.StartTime)
	profileData := []byte(profile)

	err = s.Indexer.Queue.Publish(context.Background(), indexer.Task{
		Path:      profileFileName,
		Service:   serviceNameHeader,
		Timestamp: time.Now(),
	}, profileData)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to publish profile"})
		return
	}

	c.JSON(http.StatusOK, modelresponse.ProfileResponse{
		Message: "ok",
		GPID:    gpid,
	})
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func parseGPID(raw json.RawMessage) (int64, error) {
	var asNumber int64
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		return asNumber, nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		if asString == "" {
			return 0, nil
		}
		return strconv.ParseInt(asString, 10, 64)
	}
	return 0, errors.New("invalid gpid format")
}

func getProfileFileName(startTime time.Time) string {
	startTimeStr := startTime.UTC().Format("2006-01-02T15:04:05")
	randomSuffix := randomHex(8)
	return fmt.Sprintf("%s_%s", startTimeStr, randomSuffix)
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		for i := range buf {
			buf[i] = byte(mathrand.Intn(256))
		}
	}
	return hex.EncodeToString(buf)
}
