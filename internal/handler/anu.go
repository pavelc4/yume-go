package handler

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var anuUserPref = struct {
	m map[int64]bool
}{m: make(map[int64]bool)}

func HandleAnuToggleUser(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	uid := msg.From.ID
	cur := anuUserPref.m[uid]
	anuUserPref.m[uid] = !cur

	status := "OFF"
	if anuUserPref.m[uid] {
		status = "ON"
	}
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Your anu mode is now "+status+"."))
}

func IsUserAnuEnabled(uid int64) bool {
	return anuUserPref.m[uid]
}
