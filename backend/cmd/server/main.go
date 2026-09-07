package main

import (
	"context"
	"log"
	"net/http"

	"ytagent/backend/internal/api"
	"ytagent/backend/internal/config"
	"ytagent/backend/internal/db"
	"ytagent/backend/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()
	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer database.Close()

	store := db.NewStore(database)
	settingsStore := db.NewSettingsStore(database)

	if err := settingsStore.SeedFromEnv(ctx, cfg.SeedYouTubeAPIKey, cfg.SeedDeepSeekAPIKey, cfg.SeedAIModel, cfg.SeedMaxVideos); err != nil {
		log.Fatalf("settings seed error: %v", err)
	}

	w := worker.New(store, settingsStore)
	handler := api.NewHandler(store, settingsStore, w)
	settingsHandler := api.NewSettingsHandler(settingsStore)
	router := api.NewRouter(handler, settingsHandler, cfg.CORSOrigin)

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal(err)
	}
}
