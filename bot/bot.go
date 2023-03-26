package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	ombi "github.com/kirmanak/ombibot-go/ombi"
	"log"
)

type Bot struct {
	tgbot      *tgbotapi.BotAPI
	ombiClient *ombi.OmbiClient
}

func NewBot(tgbot *tgbotapi.BotAPI, ombiClient *ombi.OmbiClient) *Bot {
	return &Bot{
		tgbot:      tgbot,
		ombiClient: ombiClient,
	}
}

func (bot *Bot) Start(fromUpdateId int) {
	u := tgbotapi.NewUpdate(fromUpdateId)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"}

	updates := bot.tgbot.GetUpdatesChan(u)

	for update := range updates {
		switch {
		case update.Message != nil:
			bot.handle_message(update.Message)
		case update.CallbackQuery != nil:
			bot.handle_callback(update.CallbackQuery)
		default:
			log.Printf("Unknown update type: %+v", update)
			bot.tgbot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Please, type a name of a movie or a TV show to search for it."))
		}
	}
}

func (bot *Bot) handle_message(message *tgbotapi.Message) {
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	switch message.Text {
	case "/start":
		response := fmt.Sprintf("Hello, %s! Type a name of a movie or a TV show to search for it.", message.Chat.FirstName)
		bot.tgbot.Send(tgbotapi.NewMessage(message.Chat.ID, response))
	case "":
		bot.tgbot.Send(tgbotapi.NewMessage(message.Chat.ID, "Please, type a name of a movie or a TV show to search for it."))
	default:
		bot.handle_search_request(message)
	}
}

func (bot *Bot) handle_callback(callbackQuery *tgbotapi.CallbackQuery) {
	log.Printf("[%s] %s", callbackQuery.From.UserName, callbackQuery.Data)
}

func (bot *Bot) handle_search_request(message *tgbotapi.Message) {
	log.Printf("Searching for %s", message.Text)
	result, err := bot.ombiClient.PerformMultiSearch(message.Text)
	if err != nil {
		log.Printf("Error while searching for %s: %s", message.Text, err)
	} else {
		log.Printf("Found %d results for %s", len(result), message.Text)
	}
}
