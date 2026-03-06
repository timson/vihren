//
// Copyright (C) 2023 Intel Corporation
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

package main

import (
	"context"

	"vihren/internal/config"
	"vihren/internal/db"
	"vihren/internal/indexer"
	"vihren/internal/logging"
	"vihren/internal/server"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

func main() {
	logging.Init()

	cfg := config.VihrenConfig{}
	if err := envconfig.Process("vihren", &cfg); err != nil {
		log.WithError(err).Fatal("failed to load config")
	}
	log.WithFields(log.Fields{
		"server_port":     cfg.Server.Port,
		"queue_path":      cfg.Queue.PayloadPath,
		"queue_depth":     cfg.Queue.QueueDepth,
		"indexer_workers": cfg.Indexer.Workers,
		"chdb_session":    cfg.DB.Filename,
	}).Info("config loaded")

	queue, err := indexer.NewQueue(cfg.Queue)
	if err != nil {
		log.WithError(err).Fatal("failed to init queue")
	}

	dbClient, err := db.NewChDBClient(cfg.DB)
	if err != nil {
		log.WithError(err).Fatal("failed to init db")
	}

	profileCh := make(chan db.ProfileBlock, cfg.Indexer.Workers*2)
	idx, err := indexer.NewIndexer(cfg.Indexer, queue, profileCh)
	if err != nil {
		log.WithError(err).Fatal("failed to init indexer")
	}
	idx.Start()

	ctx := context.Background()
	go func() {
		if err := dbClient.RunProfileWriter(ctx, profileCh); err != nil {
			log.WithError(err).Fatal("profile writer stopped")
		}
	}()

	srv := server.NewServer(queue, dbClient, idx)

	if err = srv.Run(cfg.Server.Port); err != nil {
		log.WithError(err).Fatal("server stopped")
	}

}
