package handler

import (
	"fmt"
	"log"
	"time"

	"yume-go/internal/api"
	"yume-go/internal/config"
	"yume-go/internal/util"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleGacha(bot *tgbotapi.BotAPI, message *tgbotapi.Message, apiClient *api.APIClient, cfg *config.Config) {
	typing := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	bot.Send(typing)

	apiPriority := []string{cfg.APIPrimary, cfg.APISecondary, cfg.APITertiary}

	waifu, err := apiClient.FetchRandomWaifu(false, apiPriority)
	if err != nil {
		log.Printf("Error fetching waifu: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, the gacha failed. Please try again!")
		bot.Send(msg)
		return
	}

	log.Printf("Fetched waifu %s (ID: %s) from %s", waifu.Name, waifu.ImageID, waifu.Source)

	uploadAction := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatUploadPhoto)
	bot.Send(uploadAction)

	result, err := util.DownloadToTemp(waifu.URL, waifu.ImageID)
	if err != nil {
		log.Printf("Download failed: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, failed to download image. Please try again!")
		bot.Send(msg)
		return
	}
	defer util.CleanupTemp(result.FolderPath)

	caption := fmt.Sprintf("✨ You got: %s\nID: %s\nSource: %s\nSize: %.2f MB",
		waifu.Name, waifu.ImageID, waifu.Source, float64(result.FileSize)/(1024*1024))

	sendDone := make(chan error, 1)

	go func() {
		var err error

		if result.FileSize > 10*1024*1024 {
			log.Printf("File size %d bytes, sending as document", result.FileSize)
			doc := tgbotapi.NewDocument(message.Chat.ID, tgbotapi.FilePath(result.FilePath))
			doc.Caption = caption
			_, err = bot.Send(doc)
		} else {
			photo := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FilePath(result.FilePath))
			photo.Caption = caption
			_, err = bot.Send(photo)
		}

		sendDone <- err
	}()

	select {
	case err := <-sendDone:
		if err != nil {
			log.Printf("Error sending: %v", err)
			msg := tgbotapi.NewMessage(message.Chat.ID, "Failed to send image. Please try again!")
			bot.Send(msg)
			return
		}
		log.Printf("Successfully sent waifu %s (ID: %s, %.2f MB) to user %d",
			waifu.Name, waifu.ImageID, float64(result.FileSize)/(1024*1024), message.From.ID)

	case <-time.After(60 * time.Second):
		log.Printf("Send timeout for waifu %s (ID: %s)", waifu.Name, waifu.ImageID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Upload timeout. Try again!")
		bot.Send(msg)
		return
	}
}
