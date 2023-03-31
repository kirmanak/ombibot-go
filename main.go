package main

import (
	"fmt"
	"github.com/kirmanak/ombibot-go/bot"
	"github.com/kirmanak/ombibot-go/ombi"
	"github.com/kirmanak/ombibot-go/storage"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
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

	storage, err := storage.NewStorage()
	if err != nil {
		log.Panic(err)
	}

	defer storage.Close()

	tgbot.Debug = true
	log.Printf("Authorized on account %s", tgbot.Self.UserName)

	ombiClient := ombi.NewOmbiClient(configuration.OmbiUrl, configuration.OmbiKey)

	bot := bot.NewBot(tgbot, ombiClient, configuration.PosterBasePath, storage)
	bot.Start(configuration.UpdateId)
}

// Parse configuration from environment variables
func parse_configuration() (*Configuration, error) {
	apiToken := os.Getenv("TELEGRAM_APITOKEN")
	if apiToken == "" {
		return nil, fmt.Errorf("TELEGRAM_APITOKEN is not set")
	}

	update_id := os.Getenv("TELEGRAM_UPDATE_ID")
	if update_id == "" {
		return nil, fmt.Errorf("TELEGRAM_UPDATE_ID is not set")
	}

	update_id_int, err := strconv.Atoi(update_id)
	if err != nil {
		return nil, fmt.Errorf("TELEGRAM_UPDATE_ID is not an integer: %w", err)
	}

	ombi_url := os.Getenv("OMBI_URL")
	if ombi_url == "" {
		return nil, fmt.Errorf("OMBI_URL is not set")
	}

	ombi_key := os.Getenv("OMBI_KEY")
	if ombi_key == "" {
		return nil, fmt.Errorf("OMBI_KEY is not set")
	}

	poster_base_path := os.Getenv("POSTER_BASE_PATH")
	if poster_base_path == "" {
		return nil, fmt.Errorf("POSTER_BASE_PATH is not set")
	}

	configuration := Configuration{
		ApiToken:       apiToken,
		UpdateId:       update_id_int,
		OmbiUrl:        ombi_url,
		OmbiKey:        ombi_key,
		PosterBasePath: poster_base_path,
	}

	log.Printf("Configuration: %+v", configuration)

	return &configuration, nil
}

type Configuration struct {
	ApiToken       string
	UpdateId       int
	OmbiUrl        string
	OmbiKey        string
	PosterBasePath string
}
