package bot

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/kirmanak/ombibot-go/app/ombi"
	"github.com/kirmanak/ombibot-go/app/storage"
	"log"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	previous InlineButtonType = iota
	request
	next
)

type Bot struct {
	tgbot          *tgbotapi.BotAPI
	ombiClients    map[int64]ombi.OmbiClient
	posterBasePath string
	storage        storage.Storage
}

func NewBot(tgbot *tgbotapi.BotAPI, ombiClients map[int64]ombi.OmbiClient, posterBasePath string, storage storage.Storage) *Bot {
	return &Bot{
		tgbot:          tgbot,
		ombiClients:    ombiClients,
		posterBasePath: posterBasePath,
		storage:        storage,
	}
}

func (bot *Bot) Start(fromUpdateId int) {
	u := tgbotapi.NewUpdate(fromUpdateId)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"}

	updates := bot.tgbot.GetUpdatesChan(u)

	for update := range updates {
		go bot.handle_update(update)
	}
}

func (bot *Bot) handle_update(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic: %s", r)
		}
	}()

	var chatID int64
	var response tgbotapi.Chattable
	var err error
	switch {
	case update.Message != nil:
		chatID = update.Message.Chat.ID
		response, err = bot.handle_message(update.Message)
	case update.CallbackQuery != nil:
		chatID = update.CallbackQuery.Message.Chat.ID
		response, err = bot.handle_callback(update.CallbackQuery)
	default:
		log.Printf("Unknown update type: %+v", update)
		return
	}

	bot.send_response(chatID, response, err)
}

func (bot *Bot) send_response(chatID int64, response tgbotapi.Chattable, err error) {
	if err != nil {
		log.Printf("Error: %s", err)
		response = tgbotapi.NewMessage(chatID, "There was an error: "+err.Error())
	}

	if _, err = bot.tgbot.Send(response); err != nil {
		log.Printf("Error: %s", fmt.Errorf("failed to send response: %w", err))
	}
}

func (bot *Bot) handle_message(message *tgbotapi.Message) (tgbotapi.Chattable, error) {
	log.Printf("[%s] %s", message.From.UserName, message.Text)

	switch message.Text {
	case "/start":
		response := fmt.Sprintf("Hello, %s! Type a name of a movie or a TV show to search for it.", message.Chat.FirstName)
		return tgbotapi.NewMessage(message.Chat.ID, response), nil
	case "":
		return tgbotapi.NewMessage(message.Chat.ID, "Please, type a name of a movie or a TV show to search for it."), nil
	default:
		return bot.handle_search_request(message)
	}
}

func (bot *Bot) handle_callback(callbackQuery *tgbotapi.CallbackQuery) (tgbotapi.Chattable, error) {
	log.Printf("[%s] %s", callbackQuery.From.UserName, callbackQuery.Data)

	bot.tgbot.Send(tgbotapi.NewCallback(callbackQuery.ID, ""))

	var inline_button_data InlineButtonData
	if err := json.Unmarshal([]byte(callbackQuery.Data), &inline_button_data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal inline button data: %w", err)
	}

	results_json, err := bot.storage.GetSearchResults(inline_button_data.ResultsUuid.String())
	if err != nil {
		return nil, err
	}

	var results []ombi.MultiSearchResult
	if err = json.Unmarshal([]byte(results_json), &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	switch inline_button_data.InlineButtonType {
	case previous:
		return bot.show_new_result(callbackQuery.Message, results, inline_button_data.Index-1, inline_button_data.ResultsUuid)
	case next:
		return bot.show_new_result(callbackQuery.Message, results, inline_button_data.Index+1, inline_button_data.ResultsUuid)
	default:
		return bot.request_media(callbackQuery, results[inline_button_data.Index])
	}
}

func (bot *Bot) request_media(callbackQuery *tgbotapi.CallbackQuery, result ombi.MultiSearchResult) (tgbotapi.Chattable, error) {
	log.Printf("Requesting %s", result.Title)

	client, err := bot.ombi_client_by_user(callbackQuery.From)
	if err != nil {
		return nil, err
	}

	if err := client.RequestMedia(result); err != nil {
		return nil, err
	}

	return tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Successfully requested "+result.Title), nil
}

func (bot *Bot) show_new_result(message *tgbotapi.Message, results []ombi.MultiSearchResult, index int, results_uuid uuid.UUID) (tgbotapi.Chattable, error) {
	results_size := len(results)
	real_index := (results_size + index) % results_size
	result := results[real_index]

	photoReader, err := bot.load_poster(result.Poster)
	if err != nil {
		return nil, err
	}

	photo := tgbotapi.NewInputMediaPhoto(photoReader)
	photo.Caption = caption(result)
	photo.CaptionEntities = caption_entities(result)
	msg := tgbotapi.EditMessageMediaConfig{
		BaseEdit: tgbotapi.BaseEdit{
			ChatID:    message.Chat.ID,
			MessageID: message.MessageID,
		},
		Media: photo,
	}

	markup, err := create_reply_markup(real_index, results_uuid, len(results))
	if err != nil {
		return nil, err
	}

	msg.ReplyMarkup = markup

	return msg, nil
}

func (bot *Bot) ombi_client_by_user(user *tgbotapi.User) (ombi.OmbiClient, error) {
	client := bot.ombiClients[user.ID]
	if client == nil {
		return nil, fmt.Errorf("no Ombi client for user %s", user.UserName)
	}
	return client, nil
}

func (bot *Bot) handle_search_request(message *tgbotapi.Message) (tgbotapi.Chattable, error) {
	log.Printf("Searching for %s", message.Text)

	client, err := bot.ombi_client_by_user(message.From)
	if err != nil {
		return nil, err
	}

	result, err := client.PerformMultiSearch(message.Text)
	if err != nil {
		return nil, err
	}

	log.Printf("Found results for %s: %+v", message.Text, result)

	var filtered_result []ombi.MultiSearchResult
	for _, r := range result {
		if r.Poster != "" {
			filtered_result = append(filtered_result, r)
		}
	}
	if len(filtered_result) == 0 {
		return tgbotapi.NewMessage(message.Chat.ID, "No results found."), nil
	}

	results_json, err := json.Marshal(filtered_result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search results: %w", err)
	}

	results_uuid := uuid.New()

	if err = bot.storage.SaveSearchResults(results_uuid.String(), string(results_json)); err != nil {
		return nil, err
	}

	photoReader, err := bot.load_poster(filtered_result[0].Poster)
	if err != nil {
		return nil, err
	}

	markup, err := create_reply_markup(0, results_uuid, len(filtered_result))
	if err != nil {
		return nil, err
	}

	photo_message := tgbotapi.NewPhoto(message.Chat.ID, photoReader)
	photo_message.Caption = caption(filtered_result[0])
	photo_message.CaptionEntities = caption_entities(filtered_result[0])
	photo_message.ReplyMarkup = markup

	return photo_message, nil
}

func create_reply_markup(index int, results_uuid uuid.UUID, results_size int) (*tgbotapi.InlineKeyboardMarkup, error) {
	var inline_keyboard_row []tgbotapi.InlineKeyboardButton
	if results_size > 1 {
		data, err := new_inline_button_data(previous, index, results_uuid)
		if err != nil {
			return nil, err
		}
		var previous_index int
		if index == 0 {
			previous_index = results_size
		} else {
			previous_index = index
		}
		message := fmt.Sprintf("Previous [%d/%d]", previous_index, results_size)
		inline_keyboard_row = append(inline_keyboard_row, tgbotapi.NewInlineKeyboardButtonData(message, data))
	}

	data, err := new_inline_button_data(request, index, results_uuid)
	if err != nil {
		return nil, err
	}
	var message string
	if results_size == 1 {
		message = "Request"
	} else {
		message = fmt.Sprintf("Request [%d/%d]", index+1, results_size)
	}
	inline_keyboard_row = append(inline_keyboard_row, tgbotapi.NewInlineKeyboardButtonData(message, data))

	if results_size > 1 {
		data, err := new_inline_button_data(next, index, results_uuid)
		if err != nil {
			return nil, err
		}
		var next_index int
		if index == results_size-1 {
			next_index = 1
		} else {
			next_index = index + 2
		}
		message := fmt.Sprintf("Next [%d/%d]", next_index, results_size)
		inline_keyboard_row = append(inline_keyboard_row, tgbotapi.NewInlineKeyboardButtonData(message, data))
	}

	markup := tgbotapi.NewInlineKeyboardMarkup(inline_keyboard_row)
	return &markup, nil
}

func (bot *Bot) load_poster(posterPath string) (*tgbotapi.FileReader, error) {
	url := bot.posterBasePath + posterPath
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error while loading poster %s: %w", url, err)
	}

	return &tgbotapi.FileReader{
		Name:   posterPath,
		Reader: resp.Body,
	}, nil
}

func new_inline_button_data(inlineButtonType InlineButtonType, index int, results_uuid uuid.UUID) (string, error) {
	data := InlineButtonData{
		InlineButtonType: inlineButtonType,
		Index:            index,
		ResultsUuid:      results_uuid,
	}
	json, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal inline button data: %w", err)
	}
	return string(json), nil
}

func caption(result ombi.MultiSearchResult) string {
	return fmt.Sprintf("%s\n%s", result.Title, result.Overview)
}

func caption_entities(result ombi.MultiSearchResult) []tgbotapi.MessageEntity {
	return []tgbotapi.MessageEntity{
		{
			Type:   "bold",
			Offset: 0,
			Length: len(result.Title),
		},
	}
}

type InlineButtonData struct {
	InlineButtonType InlineButtonType `json:"t"`
	Index            int              `json:"i"`
	ResultsUuid      uuid.UUID        `json:"r"`
}

type InlineButtonType int
