package handler

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var anuUserPref = struct {
	m map[int64]bool
}{m: make(map[int64]bool)}

func HandleAnuToggleUser(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	userID := msg.From.ID
	args := strings.TrimSpace(msg.CommandArguments())

	if args == "status" {
		if anuUserPref.m[userID] {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🤨"))
		} else {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "😇"))
		}
		return
	}

	cur := anuUserPref.m[userID]
	anuUserPref.m[userID] = !cur

	if anuUserPref.m[userID] {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🤨"))
	} else {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "😇"))
	}
}

func IsUserAnuEnabled(uid int64) bool {
	return anuUserPref.m[uid]
}
