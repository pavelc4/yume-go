package handler

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleStart(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	typing := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	bot.Send(typing)

	text := "Welcome to Yume-Go! 🌸\n\n" +
		"A waifu gacha bot where you can get a random waifu.\n\n" +
		"Type /help to see the list of available commands."

	buttons := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(" Developer", "https://t.me/pavellc"),
			tgbotapi.NewInlineKeyboardButtonURL(" Source Code", "https://github.com/pavelc4/yume-go"),
		),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyMarkup = buttons

	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending start message: %v", err)
	} else {
		log.Printf("Sent start message to user %d", message.From.ID)
	}
}

func HandleHelp(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	typing := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	bot.Send(typing)

	text := "📖 Command List:\n\n" +
		"/start - Start the bot\n" +
		"/help - Show this help menu\n" +
		"/gacha - Get a random waifu\n" +
		"/anu - Toggle anu\n" +
		"/profile - View your profile (coming soon)\n" +
		"/collection - View your waifu collection (coming soon)"

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending help message: %v", err)
	} else {
		log.Printf("Sent help message to user %d", message.From.ID)
	}
}
