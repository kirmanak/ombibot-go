package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	ombi "github.com/kirmanak/ombibot-go/ombi"
	bot "github.com/kirmanak/ombibot-go/bot"
)

func main() {
	configuration, err := parse_configuration()
	if err != nil {
		log.Panic(err)
	}

	tgbot, err := tgbotapi.NewBotAPI(configuration.ApiToken)
	if err != nil {
		log.Panic(err)
	}

	tgbot.Debug = true
	log.Printf("Authorized on account %s", tgbot.Self.UserName)

	ombiClient := ombi.NewOmbiClient(configuration.OmbiUrl, configuration.OmbiKey)

	bot := bot.NewBot(tgbot, ombiClient)
	bot.Start(configuration.UpdateId)
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

	ombi_url := os.Getenv("OMBI_URL")
	if ombi_url == "" {
		return Configuration{ApiToken: apiToken, UpdateId: update_id_int}, errors.New("OMBI_URL is not set")
	}

	ombi_key := os.Getenv("OMBI_KEY")
	if ombi_key == "" {
		return Configuration{ApiToken: apiToken, UpdateId: update_id_int, OmbiUrl: ombi_url}, errors.New("OMBI_KEY is not set")
	}

	configuration := Configuration{
		ApiToken: apiToken,
		UpdateId: update_id_int,
		OmbiUrl: ombi_url,
		OmbiKey: ombi_key,
	}

	log.Printf("Configuration: %+v", configuration)

	return configuration, nil
}

type Configuration struct {
	ApiToken string
	UpdateId int
	OmbiUrl  string
	OmbiKey  string
}
