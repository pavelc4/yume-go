package config

import (
	"os"
)

type Config struct {
	BotToken     string
	APIPrimary   string
	APISecondary string
	APITertiary  string
	WaifuImURL   string
	WaifuPicsURL string
	WaifuItURL   string
	WaifuWeights string
}

func Load() *Config {
	return &Config{
		BotToken:     getEnv("TELEGRAM_BOT_TOKEN", ""),
		APIPrimary:   getEnv("WAIFU_API_PRIMARY", "waifu.im"),
		APISecondary: getEnv("WAIFU_API_SECONDARY", "waifu.pics"),
		APITertiary:  getEnv("WAIFU_API_TERTIARY", "waifu.it"),
		WaifuImURL:   getEnv("WAIFU_IM_URL", "https://api.waifu.im/search"),
		WaifuPicsURL: getEnv("WAIFU_PICS_URL", "https://api.waifu.pics"),
		WaifuItURL:   getEnv("WAIFU_IT_URL", "https://waifu.it/api/v4"),
		WaifuWeights: getEnv("WAIFU_WEIGHTS", "waifu.im:1,waifu.pics:1,waifu.it:1"),
	}

}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
