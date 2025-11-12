package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"yume-go/internal/api"
	"yume-go/internal/bot"
	"yume-go/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var (
	firstHealthHost string
	firstHostOnce   sync.Once
)

func main() {
	log.Println("Yume-Go Bot starting...")
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	cfg := config.Load()
	if cfg.BotToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	telegramBot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatal("Failed to initialize bot:", err)
	}
	log.Printf("Authorized on account @%s", telegramBot.Self.UserName)

	apiClient := api.NewAPIClient(cfg.WaifuImURL, cfg.WaifuPicsURL, cfg.WaifuItURL)
	log.Println("API Client initialized")
	log.Printf("Priority: %s -> %s -> %s", cfg.APIPrimary, cfg.APISecondary, cfg.APITertiary)

	go startHealthCheck()

	time.AfterFunc(2*time.Second, func() {
		url := resolveKeepaliveURL()
		startKeepAlive(url, 4*time.Minute)
	})

	router := bot.NewRouter(telegramBot, apiClient, cfg)
	router.Start()
}

func startHealthCheck() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		captureHost(r)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Yume-Go Bot is running!"))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		captureHost(r)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Health check server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Health check server failed: %v", err)
	}
}

func captureHost(r *http.Request) {
	firstHostOnce.Do(func() {
		scheme := r.Header.Get("X-Forwarded-Proto")
		host := r.Header.Get("X-Forwarded-Host")
		if scheme == "" {
			if r.TLS != nil {
				scheme = "https"
			} else {
				scheme = "http"
			}
		}
		if host == "" {
			host = r.Host
		}
		firstHealthHost = fmt.Sprintf("%s://%s", scheme, host)
		log.Printf("Detected public host: %s", firstHealthHost)
	})
}

func resolveKeepaliveURL() string {

	if base := os.Getenv("APP_BASE_URL"); base != "" {
		return strings.TrimRight(base, "/") + "/health"
	}

	if firstHealthHost != "" {
		return firstHealthHost + "/health"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	return "http://localhost:" + port + "/health"
}

func startKeepAlive(url string, interval time.Duration) {
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		log.Printf("Keepalive enabled, pinging %s every %s", url, interval)
		for range ticker.C {
			resp, err := client.Get(url)
			if err != nil {
				log.Printf("keepalive error: %v", err)
				continue
			}
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Printf("keepalive non-200: %d", resp.StatusCode)
			}

			if firstHealthHost != "" && !strings.HasPrefix(url, firstHealthHost) {
				newURL := firstHealthHost + "/health"
				log.Printf("Switching keepalive target to %s", newURL)
				url = newURL
			}
		}
	}()
}
