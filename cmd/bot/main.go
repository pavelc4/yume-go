package main

import (
	"log"
	"net/http"
	"os"

	"yume-go/internal/api"
	"yume-go/internal/bot"
	"yume-go/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Yume-Go Bot starting...")
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	cfg := config.Load()
	if cfg.BotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	telegramBot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatal("Failed to initialize bot:", err)
	}

	log.Printf("Authorized on account @%s", telegramBot.Self.UserName)

	apiClient := api.NewAPIClient(cfg.WaifuImURL, cfg.WaifuPicsURL, cfg.WaifuItURL)
	log.Println("API Client initialized")
	log.Printf("Priority: %s -> %s -> %s", cfg.APIPrimary, cfg.APISecondary, cfg.APITertiary)

	go startHealthCheck()

	router := bot.NewRouter(telegramBot, apiClient, cfg)
	router.Start()
}

func startHealthCheck() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Yume-Go Bot is running!"))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Health check server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Health check server failed: %v", err)
	}
}
