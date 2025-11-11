package handler

import (
	"fmt"
	"log"

	"yume-go/internal/api"
	"yume-go/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleGacha(bot *tgbotapi.BotAPI, message *tgbotapi.Message, apiClient *api.APIClient, cfg *config.Config) {
	apiPriority := []string{cfg.APIPrimary, cfg.APISecondary, cfg.APITertiary}

	waifu, err := apiClient.FetchRandomWaifu(false, apiPriority)
	if err != nil {
		log.Printf("Error fetching waifu: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, the gacha failed. Please try again!")
		bot.Send(msg)
		return
	}

	photo := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FileURL(waifu.URL))
	photo.Caption = fmt.Sprintf("✨ You got: %s\nID: %s\nSource: %s",
		waifu.Name, waifu.ImageID, waifu.Source)

	_, err = bot.Send(photo)
	if err != nil {
		log.Printf("Error sending photo: %v", err)
	}
}
