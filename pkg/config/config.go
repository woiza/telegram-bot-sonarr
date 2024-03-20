package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// BotConfig ...
type Config struct {
	TelegramBotToken string
	AllowedChatIDs   map[int64]bool
	MaxItems         int
	SonarrProtocol   string
	SonarrHostname   string
	SonarrPort       int
	SonarrAPIKey     string
}

func LoadConfig() (Config, error) {
	var config Config

	config.TelegramBotToken = os.Getenv("SBOT_TELEGRAM_BOT_TOKEN")
	allowedUserIDs := os.Getenv("SBOT_BOT_ALLOWED_USERIDS")
	botMaxItems := os.Getenv("SBOT_BOT_MAX_ITEMS")
	config.SonarrProtocol = os.Getenv("SBOT_SONARR_PROTOCOL")
	config.SonarrHostname = os.Getenv("SBOT_SONARR_HOSTNAME")
	sonarrPort := os.Getenv("SBOT_SONARR_PORT")
	config.SonarrAPIKey = os.Getenv("SBOT_SONARR_API_KEY")

	// Validate required fields
	if config.TelegramBotToken == "" {
		return config, errors.New("SBOT_TELEGRAM_BOT_TOKEN is empty or not set")
	}
	if allowedUserIDs == "" {
		return config, errors.New("SBOT_BOT_ALLOWED_USERIDS is empty or not set")
	}
	if botMaxItems == "" {
		return config, errors.New("SBOT_BOT_MAX_ITEMS is empty or not set")
	}
	// Normalize and validate SBOT_SONARR_PROTOCOL
	config.SonarrProtocol = strings.ToLower(config.SonarrProtocol)
	if config.SonarrProtocol != "http" && config.SonarrProtocol != "https" {
		return config, errors.New("SBOT_SONARR_PROTOCOL must be http or https")
	}
	if config.SonarrHostname == "" {
		return config, errors.New("SBOT_SONARR_HOSTNAME is empty or not set")
	}
	if sonarrPort == "" {
		return config, errors.New("SBOT_SONARR_PORT is empty or not set")
	}
	if config.SonarrAPIKey == "" {
		return config, errors.New("SBOT_SONARR_API_KEY is empty or not set")
	}

	// Parsing SBOT_BOT_MAX_ITEMS as a number
	maxItems, err := strconv.Atoi(botMaxItems)
	if err != nil {
		return config, errors.New("SBOT_BOT_MAX_ITEMS is not a valid number")
	}
	config.MaxItems = maxItems

	// Parsing SBOT_BOT_ALLOWED_USERIDS as a list of integers
	userIDs := strings.Split(allowedUserIDs, ",")
	parsedUserIDs := make(map[int64]bool)
	for _, id := range userIDs {
		parsedID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return config, fmt.Errorf("SBOT_BOT_ALLOWED_USERIDS contains non-integer value: %s", err)
		}
		parsedUserIDs[parsedID] = true
	}
	config.AllowedChatIDs = parsedUserIDs

	// Parsing SBOT_SONARR_PORT as a number
	port, err := strconv.Atoi(sonarrPort)
	if err != nil {
		return config, errors.New("SBOT_SONARR_PORT is not a valid number")
	}
	config.SonarrPort = port

	return config, nil
}
