package bot

import (
	"log"

	"yume-go/internal/api"
	"yume-go/internal/config"
	"yume-go/internal/handler"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Router struct {
	bot       *tgbotapi.BotAPI
	apiClient *api.APIClient
	config    *config.Config
}

func NewRouter(bot *tgbotapi.BotAPI, apiClient *api.APIClient, cfg *config.Config) *Router {
	return &Router{
		bot:       bot,
		apiClient: apiClient,
		config:    cfg,
	}
}

func (r *Router) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := r.bot.GetUpdatesChan(u)

	log.Println("Bot is running. Press CTRL+C to stop.")

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			r.handleCommand(update.Message)
		}
	}
}

func (r *Router) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		handler.HandleStart(r.bot, message)

	case "help":
		handler.HandleHelp(r.bot, message)

	case "gacha":
		handler.HandleGacha(r.bot, message, r.apiClient, r.config)

	default:
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"Command tidak dikenal. Ketik /help untuk bantuan.")
		r.bot.Send(msg)
	}
}
