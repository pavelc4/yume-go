package bot

import (
	"log"
	"strings"
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

func normalizeCommand(text, botUsername string) string {
	text = strings.TrimLeft(text, " ")

	atPos := strings.IndexByte(text, '@')
	if atPos == -1 {
		return text
	}
	command := text[:atPos]
	rest := text[atPos+1:]

	mentionEnd := strings.IndexByte(rest, ' ')
	if mentionEnd == -1 {
		mentionEnd = len(rest)
	}
	usernamePart := rest[:mentionEnd]

	if strings.ToLower(usernamePart) != strings.ToLower(botUsername) {
		return text
	}

	remaining := rest[mentionEnd:]
	if len(remaining) == 0 {
		return command
	}
	return command + remaining
}

func parseCommand(text string) (cmd string, ok bool) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", false
	}
	space := strings.IndexByte(text, ' ')
	if space == -1 {
		space = len(text)
	}
	raw := text[1:space]
	if strings.IndexByte(raw, '@') != -1 {
		return "", false
	}
	return strings.ToLower(raw), true
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

		original := update.Message.Text
		botUsername := r.bot.Self.UserName
		normalized := normalizeCommand(original, botUsername)
		cmd, ok := parseCommand(normalized)
		if !ok {
			continue
		}

		log.Printf("Message from @%s (ID: %d): %s",
			update.Message.From.UserName,
			update.Message.From.ID,
			original)

		if handlerFunc, exists := r.commands[cmd]; exists {
			r.wg.Add(1)
			go func(msg *tgbotapi.Message, hf func(*tgbotapi.BotAPI, *tgbotapi.Message)) {
				defer r.wg.Done()
				hf(r.bot, msg)
			}(update.Message, handlerFunc)
		} else {
			r.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command. Type /help for assistance."))
		}
	}
}
