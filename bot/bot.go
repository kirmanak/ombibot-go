package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/kirmanak/ombibot-go/ombi"
	"log"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"
)

const (
	previous InlineButtonType = iota
	request
	next
)

type Bot struct {
	tgbot          *tgbotapi.BotAPI
	ombiClient     *ombi.OmbiClient
	posterBasePath string
	db             *sql.DB
}

func NewBot(tgbot *tgbotapi.BotAPI, ombiClient *ombi.OmbiClient, posterBasePath string, db *sql.DB) *Bot {
	return &Bot{
		tgbot:          tgbot,
		ombiClient:     ombiClient,
		posterBasePath: posterBasePath,
		db:             db,
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
	bot.tgbot.Send(tgbotapi.NewCallback(callbackQuery.ID, ""))
	var inline_button_data InlineButtonData
	json.Unmarshal([]byte(callbackQuery.Data), &inline_button_data)

	rows, err := bot.db.Query("SELECT results FROM search_results WHERE uuid = ?", inline_button_data.ResultsUuid.String())
	if err != nil {
		log.Printf("Error while querying database: %s", err)
		bot.tgbot.Send(tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Something went wrong while processing request. Please, try again later."))
		return
	}
	defer rows.Close()

	if !rows.Next() {
		log.Printf("No results found for uuid %s", inline_button_data.ResultsUuid.String())
		bot.tgbot.Send(tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Something went wrong while processing request. Please, try again later."))
		return
	}

	var results_json string
	err = rows.Scan(&results_json)
	if err != nil {
		log.Printf("Error while scanning database row: %s", err)
		bot.tgbot.Send(tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Something went wrong while processing request. Please, try again later."))
		return
	}

	var results []ombi.MultiSearchResult
	json.Unmarshal([]byte(results_json), &results)

	switch inline_button_data.InlineButtonType {
	case previous:
		bot.show_new_result(callbackQuery.Message, results, inline_button_data.Index-1, inline_button_data.ResultsUuid)
	case next:
		bot.show_new_result(callbackQuery.Message, results, inline_button_data.Index+1, inline_button_data.ResultsUuid)
	case request:
		return
	}
}

func (bot *Bot) show_new_result(message *tgbotapi.Message, results []ombi.MultiSearchResult, index int, results_uuid uuid.UUID) {
	result := results[index]
	caption := fmt.Sprintf("%s\n%s", result.Title, result.Overview)
	var inline_keyboard_row []tgbotapi.InlineKeyboardButton
	if index > 0 {
		inline_keyboard_row = append(inline_keyboard_row,
			tgbotapi.NewInlineKeyboardButtonData("Previous", new_inline_button_data(previous, index, results_uuid)),
		)
	}
	inline_keyboard_row = append(inline_keyboard_row,
		tgbotapi.NewInlineKeyboardButtonData("Request", new_inline_button_data(request, index, results_uuid)),
	)
	if index < len(results)-1 {
		inline_keyboard_row = append(inline_keyboard_row,
			tgbotapi.NewInlineKeyboardButtonData("Next", new_inline_button_data(next, index, results_uuid)),
		)
	}
	photoReader, err := bot.load_poster(result.Poster)
	if err != nil {
		log.Printf("Error while loading poster: %s", err)
		bot.tgbot.Send(tgbotapi.NewMessage(message.Chat.ID, "Something went wrong while loading poster. Please, try again later."))
		return
	}

	photo := tgbotapi.NewInputMediaPhoto(photoReader)
	photo.Caption = caption
	msg := tgbotapi.EditMessageMediaConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID:    message.Chat.ID,
			MessageID: message.MessageID,
		},
		Media: photo,
	}
	markup := tgbotapi.NewInlineKeyboardMarkup(inline_keyboard_row)
	msg.ReplyMarkup = &markup
	bot.tgbot.Send(msg)
}

func (bot *Bot) handle_search_request(message *tgbotapi.Message) {
	log.Printf("Searching for %s", message.Text)
	result, err := bot.ombiClient.PerformMultiSearch(message.Text)
	if err != nil {
		log.Printf("Error while searching for %s: %s", message.Text, err)
		bot.tgbot.Send(tgbotapi.NewMessage(message.Chat.ID, "Something went wrong while searching for your request. Please, try again later."))
		return
	}

	log.Printf("Found results for %s: %+v", message.Text, result)

	var filtered_result []ombi.MultiSearchResult
	for _, r := range result {
		if r.Poster != "" {
			filtered_result = append(filtered_result, r)
		}
	}
	if len(filtered_result) == 0 {
		bot.tgbot.Send(tgbotapi.NewMessage(message.Chat.ID, "No results found."))
		return
	}

	results_json, _ := json.Marshal(filtered_result)
	results_uuid := uuid.New()

	_, err = bot.db.Exec("INSERT INTO search_results (uuid, results) VALUES (?, ?)", results_uuid.String(), string(results_json))
	if err != nil {
		log.Printf("Error while inserting search results into database: %s", err)
		bot.tgbot.Send(tgbotapi.NewMessage(message.Chat.ID, "Something went wrong while processing request. Please, try again later."))
		return
	}

	photoReader, err := bot.load_poster(filtered_result[0].Poster)
	if err != nil {
		log.Printf("Error while loading poster %s: %s", filtered_result[0].Poster, err)
		bot.tgbot.Send(tgbotapi.NewMessage(message.Chat.ID, "Something went wrong while loading poster. Please, try again later."))
		return
	}

	var inline_buttons_row []tgbotapi.InlineKeyboardButton

	request_button_data := new_inline_button_data(request, 0, results_uuid)
	inline_buttons_row = append(inline_buttons_row, tgbotapi.NewInlineKeyboardButtonData("Request", request_button_data))

	if len(filtered_result) > 1 {
		next_button_data := new_inline_button_data(next, 0, results_uuid)
		inline_buttons_row = append(inline_buttons_row, tgbotapi.NewInlineKeyboardButtonData("Next", next_button_data))
	}

	photo_message := tgbotapi.NewPhoto(message.Chat.ID, photoReader)
	photo_message.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(inline_buttons_row)
	bot.tgbot.Send(photo_message)
}

func (bot *Bot) load_poster(posterPath string) (*tgbotapi.FileReader, error) {
	url := bot.posterBasePath + posterPath
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error while loading poster %s: %s", url, err)
	}

	return &tgbotapi.FileReader{
		Name:   posterPath,
		Reader: resp.Body,
	}, nil
}

func new_inline_button_data(inlineButtonType InlineButtonType, index int, results_uuid uuid.UUID) string {
	data := InlineButtonData{
		InlineButtonType: inlineButtonType,
		Index:            index,
		ResultsUuid:      results_uuid,
	}
	json, _ := json.Marshal(data)
	return string(json)
}

type InlineButtonData struct {
	InlineButtonType InlineButtonType `json:"t"`
	Index            int              `json:"i"`
	ResultsUuid      uuid.UUID        `json:"r"`
}

type InlineButtonType int
