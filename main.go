package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	configuration, err := parse_configuration()
	if err != nil {
		log.Panic(err)
	}

	bot, err := tgbotapi.NewBotAPI(configuration.ApiToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(configuration.UpdateId)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		}
	}
}

// Parse configuration from environment variables
func parse_configuration() (Configuration, error) {
	apiToken := os.Getenv("TELEGRAM_APITOKEN")
	if apiToken == "" {
		return Configuration{}, errors.New("TELEGRAM_APITOKEN is not set")
	}
	update_id := os.Getenv("TELEGRAM_UPDATE_ID")
	if update_id == "" {
		return Configuration{ApiToken: apiToken}, errors.New("TELEGRAM_UPDATE_ID is not set")
	}
	update_id_int, err := strconv.Atoi(update_id)
	if err != nil {
		return Configuration{ApiToken: apiToken}, fmt.Errorf("TELEGRAM_UPDATE_ID is not an integer: %w", err)
	}

	configuration := Configuration {
		ApiToken: apiToken,
		UpdateId: update_id_int,
	}

	log.Printf("Configuration: %+v", configuration)

	return configuration, nil
}

type Configuration struct {
	ApiToken string
	UpdateId int
}
