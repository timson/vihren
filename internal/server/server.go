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
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"vihren/internal/db"
	"vihren/internal/indexer"
)

type Config struct {
	Port int `default:"8080"`
}

type Server struct {
	Queue   *indexer.Queue
	DB      *db.ChDBClient
	Indexer *indexer.Indexer
}

func NewServer(queue *indexer.Queue, dbClient *db.ChDBClient, idx *indexer.Indexer) *Server {
	log.WithFields(log.Fields{
		"queue":   queue != nil,
		"db":      dbClient != nil,
		"indexer": idx != nil,
	}).Info("server init")

	return &Server{
		Queue:   queue,
		DB:      dbClient,
		Indexer: idx,
	}
}

func (s *Server) Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(StartTime())
	router.Use(requestLogger())
	router.Use(gin.RecoveryWithWriter(log.StandardLogger().Writer()))

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	router.Use(cors.New(corsConfig))
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	router.GET("/ui", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/")
	})
	router.Static("/ui", "./ui")

	// Register endpoints
	router.GET("/api/v1/flamegraph", s.GetFlamegraph)
	router.GET("/api/v1/query", s.QueryMeta)
	router.GET("/api/v1/sessions_count", s.QuerySessionsCount)
	router.GET("/api/v1/services", s.QueryServices)
	router.GET("/api/v1/metrics/graph", s.GetMetricsGraph)
	router.GET("/api/v1/summary", s.GetSummary)
	router.GET("/api/v1/flamedb/status", s.GetDBStatus)
	router.POST("/api/v2/profiles", s.NewProfileV2)
	router.GET("/api/v1/health_check", s.HealthCheck)
	router.POST("/api/v1/logs", s.Logs)

	return router
}

func (s *Server) Run(port int) error {
	log.WithField("port", port).Info("starting server")
	return s.Router().Run(fmt.Sprintf("0.0.0.0:%d", port))
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		if rawQuery != "" {
			path = path + "?" + rawQuery
		}

		dataLength := c.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}

		log.WithFields(log.Fields{
			"statusCode": c.Writer.Status(),
			"latency":    latency.Milliseconds(),
			"clientIP":   c.ClientIP(),
			"method":     c.Request.Method,
			"path":       path,
			"dataLength": dataLength,
			"userAgent":  c.Request.UserAgent(),
		}).Debug("gin")
	}
}
