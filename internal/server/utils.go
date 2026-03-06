//
// Copyright (C) 2023 Intel Corporation
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
	"strconv"
	"strings"
	"time"

	"github.com/a8m/rql"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type filterQueryProvider interface {
	FilterQuery() string
}

type timeRangeNormalizer interface {
	NormalizeTimeRange()
}

func StartTime() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("requestStartTime", time.Now())
		c.Next()
	}
}

func parseParams[T any](params T, parser *rql.Parser, c *gin.Context) (T, string, error) {
	var query string
	var err error
	if err = c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return params, query, err
	}

	if parser != nil {
		if filterProvider, ok := any(&params).(filterQueryProvider); ok {
			rawFilterData := []byte(filterProvider.FilterQuery())
			if len(rawFilterData) > 0 { // filter parameter was passed
				query, err = buildQuery(parser, rawFilterData)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return params, query, err
				}
			}
		}
	}

	if normalizer, ok := any(&params).(timeRangeNormalizer); ok {
		normalizer.NormalizeTimeRange()
	}

	return params, query, nil
}

func buildQuery(parser *rql.Parser, rawFilterData []byte) (string, error) {
	var query string
	var expressions []string
	var args []interface{}
	filters, err := parser.Parse(rawFilterData)
	if err != nil {
		return "", err
	}
	if filters != nil {
		expressions = strings.Split(filters.FilterExp, "?")
		args = filters.FilterArgs
	}
	for idx, expr := range expressions {
		v := ""
		if idx < len(args) {
			switch args[idx].(type) {
			case int:
				v = strconv.Itoa(args[idx].(int))
			case string:
				escaped := strings.ReplaceAll(args[idx].(string), "'", "''")
				v = fmt.Sprintf(`'%s'`, escaped)
			default:
				log.WithFields(log.Fields{
					"arg":  args[idx],
					"type": fmt.Sprintf("%T", args[idx]),
				}).Warn("unable to cast filter arg")
				continue
			}
		}
		query += fmt.Sprintf("%s%s", expr, v)
	}
	// if query is nonempty add prefix AND for using inside SQL query after time range block
	if len(query) > 0 {
		query = "AND " + query
	}
	return query, nil
}
