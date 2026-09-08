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
	connectedChannelStore := db.NewConnectedChannelStore(database)
	scriptStore := db.NewScriptStore(database)

	if err := settingsStore.SeedFromEnv(ctx, cfg.SeedYouTubeAPIKey, cfg.SeedDeepSeekAPIKey, cfg.SeedAIModel, cfg.SeedMaxVideos); err != nil {
		log.Fatalf("settings seed error: %v", err)
	}

	w := worker.New(store, settingsStore)
	handler := api.NewHandler(store, settingsStore, w)
	settingsHandler := api.NewSettingsHandler(settingsStore)
	trendingHandler := api.NewTrendingHandler(settingsStore)
	oauthHandler := api.NewOAuthHandler(cfg, connectedChannelStore)
	channelsHandler := api.NewChannelsHandler(cfg, connectedChannelStore, settingsStore)
	videoPromptHandler := api.NewVideoPromptHandler(settingsStore)
	scriptHandler := api.NewScriptHandler(settingsStore, scriptStore)
	authHandler := api.NewAuthHandler(cfg.AppPassword)
	router := api.NewRouter(handler, settingsHandler, trendingHandler, oauthHandler, channelsHandler, videoPromptHandler, scriptHandler, authHandler, cfg.CORSOrigins, cfg.AppPassword)

	if cfg.AppPassword == "" {
		log.Println("warning: APP_PASSWORD is not set — the API is open to anyone who can reach it")
	}
	if cfg.GoogleClientID == "" {
		log.Println("info: Google OAuth not configured — \"Kênh của tôi\" (connected channels) will be unavailable until GOOGLE_CLIENT_ID/SECRET/REDIRECT_URL and FRONTEND_URL are set")
	}
	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal(err)
	}
}
