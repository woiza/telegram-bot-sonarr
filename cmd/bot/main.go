package main

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golift.io/starr"
	"golift.io/starr/sonarr"

	"github.com/woiza/telegram-bot-sonarr/pkg/bot"
	"github.com/woiza/telegram-bot-sonarr/pkg/config"
)

func main() {
	fmt.Println("Starting bot...")

	// get config from environment variables
	config, err := config.LoadConfig()
	if err != nil {
		// Handle error: configuration is incomplete or invalid
		log.Fatal(err)
	}

	b, err := tgbotapi.NewBotAPI(config.TelegramBotToken)
	if err != nil {
		log.Fatal("Error while starting bot: ", err)
	}

	fmt.Printf("Authorized on account %v\n", b.Self.UserName)

	sonarrConfig := starr.New(config.SonarrAPIKey, fmt.Sprintf("%v://%v:%v", config.SonarrProtocol, config.SonarrHostname, config.SonarrPort), 0)
	sonarrServer := sonarr.New(sonarrConfig)

	botInstance := bot.New(&config, b, sonarrServer)

	// Channel for receiving updates from the bot API
	updates := make(chan tgbotapi.Update)
	defer close(updates)

	// Start a goroutine to fetch updates from the bot API and send to the updates channel
	go func() {
		lastUpdateID := 0
		for {
			updateConfig := tgbotapi.NewUpdate(lastUpdateID + 1)
			updateConfig.Timeout = 60

			updatesChannel := b.GetUpdatesChan(updateConfig)
			if err != nil {
				log.Println("Error getting updates channel:", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for update := range updatesChannel {
				updates <- update // Send updates to the updates channel
				lastUpdateID = update.UpdateID
			}
		}
	}()

	// Start a goroutine to handle updates concurrently
	go botInstance.HandleUpdates(updates)

	// This can be a long-running process to handle incoming updates
	select {}
}
