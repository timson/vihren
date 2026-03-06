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

package db

import (
	"context"
	"database/sql"
	"errors"
	"vihren/internal/model/request"

	sq "github.com/Masterminds/squirrel"
	log "github.com/sirupsen/logrus"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrUnexpectedRows = errors.New("unexpected rows")
)

type QueryFunc interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func ApplyParams(qb sq.SelectBuilder, p *request.FiltersQuery, filterQuery string) (string, sq.SelectBuilder) {
	if p == nil {
		return "", qb
	}
	tbl, extraWhere := BuildConditionsSq(p, filterQuery)
	if extraWhere != nil {
		return tbl, qb.Where(extraWhere)
	}
	return tbl, qb
}

func QuerySlice[T any](
	ctx context.Context,
	q QueryFunc,
	logger *log.Entry,
	query string,
	args []any,
	scanFn func(*sql.Rows) (T, error),
) ([]T, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		if logger != nil {
			logger.WithError(err).Error("query failed")
		}
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]T, 0, 64)
	for rows.Next() {
		v, scanErr := scanFn(rows)
		if scanErr != nil {
			if logger != nil {
				logger.WithError(scanErr).Error("scan failed")
			}
			return nil, scanErr
		}
		out = append(out, v)
	}
	if err = rows.Err(); err != nil {
		if logger != nil {
			logger.WithError(err).Error("rows iteration failed")
		}
		return nil, err
	}
	return out, nil
}

func QueryRows[T any](
	ctx context.Context,
	q QueryFunc,
	logger *log.Entry,
	query string,
	args []any,
	processFn func(*sql.Rows) (T, error),
) (T, error) {
	var v T
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		if logger != nil {
			logger.WithError(err).Error("query failed")
		}
		return v, err
	}
	defer func() { _ = rows.Close() }()

	v, pErr := processFn(rows)
	return v, pErr
}

func SelectAllSq[T any](
	ctx context.Context,
	q QueryFunc,
	logger *log.Entry,
	qb sq.SelectBuilder,
	scanFn func(*sql.Rows) (T, error),
) ([]T, error) {
	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}
	return QuerySlice[T](ctx, q, logger, query, args, scanFn)
}

func SelectOneSq[T any](
	ctx context.Context,
	q QueryFunc,
	logger *log.Entry,
	qb sq.SelectBuilder,
	scanFn func(*sql.Rows) (T, error),
) (T, error) {
	var zero T

	items, err := SelectAllSq[T](ctx, q, logger, qb, scanFn)
	if err != nil {
		return zero, err
	}
	if len(items) == 0 {
		return zero, ErrNotFound
	}
	if len(items) > 1 {
		return zero, ErrUnexpectedRows
	}
	return items[0], nil
}
