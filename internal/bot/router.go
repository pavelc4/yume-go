package bot

import (
	"log"
	"sync"

	"yume-go/internal/api"
	"yume-go/internal/config"
	"yume-go/internal/handler"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Router struct {
	bot       *tgbotapi.BotAPI
	apiClient *api.APIClient
	config    *config.Config
	wg        sync.WaitGroup
	commands  map[string]func(*tgbotapi.BotAPI, *tgbotapi.Message)
}

func NewRouter(bot *tgbotapi.BotAPI, apiClient *api.APIClient, cfg *config.Config) *Router {
	r := &Router{
		bot:       bot,
		apiClient: apiClient,
		config:    cfg,
	}

	r.commands = map[string]func(*tgbotapi.BotAPI, *tgbotapi.Message){
		"start": handler.HandleStart,
		"help":  handler.HandleHelp,
		"anu":   handler.HandleAnuToggleUser,
	}

	r.commands["gacha"] = func(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
		handler.HandleGacha(bot, msg, r.apiClient, r.config)
	}

	return r
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

		log.Printf("Message from @%s (ID: %d): %s",
			update.Message.From.UserName,
			update.Message.From.ID,
			update.Message.Text)

		if update.Message.IsCommand() {
			r.wg.Add(1)
			go r.handleCommandConcurrent(update.Message)
		}
	}

	r.wg.Wait()
}

func (r *Router) handleCommandConcurrent(message *tgbotapi.Message) {
	defer r.wg.Done()

	cmd := message.Command()
	log.Printf("Processing command: /%s from user %d", cmd, message.From.ID)

	if handlerFunc, ok := r.commands[cmd]; ok {
		handlerFunc(r.bot, message)
	} else {
		r.bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Unknown command. Type /help for assistance."))
	}
}
