package handler

import (
	"fmt"
	"log"
	"strings"
	"time"

	"yume-go/internal/api"
	"yume-go/internal/config"
	"yume-go/internal/util"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func escapeHTML(s string) string {
	if s == "" {
		return s
	}
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return r.Replace(s)
}

func buildCaptionSimple(waifu *api.Waifu) string {
	char := waifu.Character
	if char == "" {
		if waifu.Name != "" {
			char = waifu.Name
		} else if len(waifu.Tags) > 0 {
			char = waifu.Tags[0]
		} else {
			char = "Unknown"
		}
	}
	charEsc := escapeHTML(char)
	idEsc := escapeHTML(waifu.ImageID)
	return fmt.Sprintf("✨ You got: <b>%s</b>\nID: %s", charEsc, idEsc)
}

func HandleGacha(bot *tgbotapi.BotAPI, message *tgbotapi.Message, apiClient *api.APIClient, cfg *config.Config) {
	typing := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	bot.Send(typing)

	apiPriority := []string{cfg.APIPrimary, cfg.APISecondary, cfg.APITertiary}

	isAnu := IsUserAnuEnabled(message.From.ID)
	waifu, err := apiClient.FetchRandomWaifu(isAnu, apiPriority)

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

	caption := buildCaptionSimple(waifu)

	sendDone := make(chan error, 1)

	go func() {
		var err error

		if result.FileSize > 10*1024*1024 {
			log.Printf("File size %d bytes, sending as document", result.FileSize)
			doc := tgbotapi.NewDocument(message.Chat.ID, tgbotapi.FilePath(result.FilePath))
			doc.Caption = caption
			doc.ParseMode = "HTML"
			_, err = bot.Send(doc)
		} else {
			photo := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FilePath(result.FilePath))
			photo.Caption = caption
			photo.ParseMode = "HTML"
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
		log.Printf("Successfully sent waifu %s (ID: %s) to user %d",
			waifu.Character, waifu.ImageID, message.From.ID)

	case <-time.After(60 * time.Second):
		log.Printf("Send timeout for waifu %s (ID: %s)", waifu.Character, waifu.ImageID)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Upload timeout. Try again!")
		bot.Send(msg)
		return
	}
}
