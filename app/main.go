package main

import (
	"fmt"
	"github.com/kirmanak/ombibot-go/app/bot"
	"github.com/kirmanak/ombibot-go/app/ombi"
	"github.com/kirmanak/ombibot-go/app/storage"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	configuration, err := parse_configuration()
	if err != nil {
		log.Fatalf("failed to parse configuration: %s", err)
	}

	storage, err := storage.NewStorage(configuration.DbPath)
	if err != nil {
		log.Fatalf("failed to open storage: %s", err)
	}

	defer storage.Close()

	users, err := storage.GetUsers()
	if err != nil {
		log.Fatalf("failed to get users: %s", err)
	}

	if len(users) == 0 {
		log.Fatalf("no users found")
	}

	tgbot, err := tgbotapi.NewBotAPI(configuration.ApiToken)
	if err != nil {
		log.Fatalf("failed to create bot: %s", err)
	}

	tgbot.Debug = true
	log.Printf("Authorized on account %s", tgbot.Self.UserName)

	ombiClients := make(map[int64]ombi.OmbiClient)
	for _, user := range users {
		ombiClient := ombi.NewOmbiClient(user.OmbiUrl, user.OmbiKey)
		ombiClients[user.Id] = ombiClient
	}

	bot := bot.NewBot(tgbot, ombiClients, configuration.PosterBasePath, storage)
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

	poster_base_path := os.Getenv("POSTER_BASE_PATH")
	if poster_base_path == "" {
		return nil, fmt.Errorf("POSTER_BASE_PATH is not set")
	}

	db_path := os.Getenv("DB_PATH")
	if db_path == "" {
		return nil, fmt.Errorf("DB_PATH is not set")
	}

	configuration := Configuration{
		ApiToken:       apiToken,
		UpdateId:       update_id_int,
		PosterBasePath: poster_base_path,
		DbPath:         db_path,
	}

	log.Printf("Configuration: %+v", configuration)

	return &configuration, nil
}

type Configuration struct {
	ApiToken       string
	UpdateId       int
	PosterBasePath string
	DbPath         string
}
