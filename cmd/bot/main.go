package main

import (
	"log"

	"yume-go/internal/api"
	"yume-go/internal/bot"
	"yume-go/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	cfg := config.Load()
	if cfg.BotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	telegramBot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatal("Failed to initialize bot: ", err)
	}

	log.Printf("Authorized on account @%s", telegramBot.Self.UserName)

	apiClient := api.NewAPIClient(cfg.WaifuImURL, cfg.WaifuPicsURL, cfg.WaifuItURL)
	log.Printf("API Client initialized")
	log.Printf("Priority: %s -> %s -> %s", cfg.APIPrimary, cfg.APISecondary, cfg.APITertiary)

	router := bot.NewRouter(telegramBot, apiClient, cfg)
	router.Start()
}
