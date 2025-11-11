package bot

import (
	"log"
	"sync"
	"time"

	"yume-go/internal/api"
	"yume-go/internal/config"
	"yume-go/internal/handler"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Router struct {
	bot          *tgbotapi.BotAPI
	apiClient    *api.APIClient
	config       *config.Config
	wg           sync.WaitGroup
	rateLimiter  map[int64]time.Time
	limiterMutex sync.RWMutex
	commands     map[string]func(*tgbotapi.BotAPI, *tgbotapi.Message)
}

func NewRouter(bot *tgbotapi.BotAPI, apiClient *api.APIClient, cfg *config.Config) *Router {
	r := &Router{
		bot:         bot,
		apiClient:   apiClient,
		config:      cfg,
		rateLimiter: make(map[int64]time.Time),
	}

	r.commands = map[string]func(*tgbotapi.BotAPI, *tgbotapi.Message){
		"start": handler.HandleStart,
		"help":  handler.HandleHelp,
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

	r.limiterMutex.Lock()
	lastRequest, exists := r.rateLimiter[message.From.ID]
	if exists && time.Since(lastRequest) < 2*time.Second {
		r.limiterMutex.Unlock()
		log.Printf("Rate limited user %d", message.From.ID)
		r.bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Wait a moment, don't spam!"))
		return
	}
	r.rateLimiter[message.From.ID] = time.Now()
	r.limiterMutex.Unlock()

	go r.cleanupRateLimiter()

	cmd := message.Command()
	log.Printf("Processing command: /%s from user %d", cmd, message.From.ID)

	if handlerFunc, ok := r.commands[cmd]; ok {
		handlerFunc(r.bot, message)
	} else {
		r.bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Unknown command. Type /help for assistance."))
	}
}

func (r *Router) cleanupRateLimiter() {
	r.limiterMutex.Lock()
	defer r.limiterMutex.Unlock()

	now := time.Now()
	for userID, lastTime := range r.rateLimiter {
		if now.Sub(lastTime) > 5*time.Minute {
			delete(r.rateLimiter, userID)
		}
	}
}
