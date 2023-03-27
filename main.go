package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"database/sql"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	bot "github.com/kirmanak/ombibot-go/bot"
	ombi "github.com/kirmanak/ombibot-go/ombi"
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

	db, err := sql.Open("sqlite3", "./ombibot.db")
	if err != nil {
		log.Panic(err)
	}

	defer db.Close()

	db.Exec("CREATE TABLE IF NOT EXISTS search_results (uuid TEXT PRIMARY KEY, results TEXT)")

	tgbot.Debug = true
	log.Printf("Authorized on account %s", tgbot.Self.UserName)

	ombiClient := ombi.NewOmbiClient(configuration.OmbiUrl, configuration.OmbiKey)
	
	bot := bot.NewBot(tgbot, ombiClient, configuration.PosterBasePath, db)
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

	poster_base_path := os.Getenv("POSTER_BASE_PATH")
	if poster_base_path == "" {
		return Configuration{ApiToken: apiToken, UpdateId: update_id_int, OmbiUrl: ombi_url, OmbiKey: ombi_key}, errors.New("POSTER_BASE_PATH is not set")
	}

	configuration := Configuration{
		ApiToken:         apiToken,
		UpdateId:         update_id_int,
		OmbiUrl:          ombi_url,
		OmbiKey:          ombi_key,
		PosterBasePath: poster_base_path,
	}

	log.Printf("Configuration: %+v", configuration)

	return configuration, nil
}

type Configuration struct {
	ApiToken       string
	UpdateId       int
	OmbiUrl        string
	OmbiKey        string
	PosterBasePath string
}