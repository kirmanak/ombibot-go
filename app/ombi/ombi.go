package ombi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

type OmbiClient interface {
	PerformMultiSearch(query string) ([]MultiSearchResult, error)
	RequestMedia(result MultiSearchResult) error
}

type SimpleOmbiClient struct {
	url string
	key string
}

func NewOmbiClient(url string, key string) *SimpleOmbiClient {
	return &SimpleOmbiClient{
		url: url,
		key: key,
	}
}

func (client *SimpleOmbiClient) PerformMultiSearch(query string) ([]MultiSearchResult, error) {
	searchRequest := &MulitSearchRequest{
		Movies:  true,
		TvShows: true,
		Music:   false,
		People:  false,
	}
	url := "api/v2/Search/multi/" + query

	var result []MultiSearchResult
	err := client.post(url, searchRequest, &result)
	if err != nil {
		return nil, fmt.Errorf("can't request %s from client %+v: %w", url, client, err)
	}

	return result, nil
}

func (client *SimpleOmbiClient) RequestMedia(result MultiSearchResult) error {
	request_body, err := create_request_body(result)
	if err != nil {
		return fmt.Errorf("can't create request body for %+v: %w", result, err)
	}

	request_path, err := get_media_request_path(result)
	if err != nil {
		return err
	}

	var response MediaRequestResult
	err = client.post(request_path, request_body, &response)
	if err != nil {
		return fmt.Errorf("can't request %s from client %+v: %w", request_path, client, err)
	}

	if response.IsError {
		return fmt.Errorf("got error response: " + response.ErrorMessage)
	}

	return nil
}

func get_media_request_path(result MultiSearchResult) (string, error) {
	if result.MediaType == "movie" {
		return "api/v1/Request/movie", nil
	} else if result.MediaType == "tv" {
		return "api/v2/Requests/tv", nil
	} else {
		return "", fmt.Errorf("unknown media type " + result.MediaType)
	}
}

func create_request_body(result MultiSearchResult) (*MediaRequest, error) {
	dbId, err := strconv.Atoi(result.Id)
	if err != nil {
		return nil, fmt.Errorf("can't convert %s to int: %w", result.Id, err)
	}

	request := &MediaRequest{
		TheMovieDbId: dbId,
	}

	return request, nil
}

func (client *SimpleOmbiClient) post(path string, request any, response any) error {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("can't encode JSON request %+v: %w", request, err)
	}

	url := fmt.Sprintf("%s/%s", client.url, path)

	httpRequest, err := http.NewRequest("POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("can't create POST request %s from client %+v: %w", url, client, err)
	}

	log.Printf("POST %s with body %s", url, string(requestBody))

	httpRequest.Header.Set("Content-Type", "application/json")
	resp, err := client.doRequest(httpRequest)
	if err != nil {
		return fmt.Errorf("can't request %s from client %+v: %w", url, client, err)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("can't read response body: %w", err)
	}

	log.Printf("Response status: %s, response body: %s", resp.Status, string(responseBody))

	if (resp.StatusCode < 200) || (resp.StatusCode >= 300) {
		return fmt.Errorf("got non-2xx response status: " + resp.Status)
	}

	if err := json.Unmarshal(responseBody, response); err != nil {
		return fmt.Errorf("can't decode JSON response: %w", err)
	}

	return nil
}

func (client *SimpleOmbiClient) doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("ApiKey", client.key)
	return http.DefaultClient.Do(req)
}

type MultiSearchResult struct {
	Id        string `json:"id"`
	MediaType string `json:"mediaType"`
	Title     string `json:"title"`
	Poster    string `json:"poster"`
	Overview  string `json:"overview"`
}

type MulitSearchRequest struct {
	Movies  bool `json:"movies"`
	TvShows bool `json:"tvShows"`
	Music   bool `json:"music"`
	People  bool `json:"people"`
}

type MediaRequestResult struct {
	Result       bool   `json:"result"`
	Message      string `json:"message"`
	IsError      bool   `json:"isError"`
	ErrorMessage string `json:"errorMessage"`
	ErrorCode    string `json:"errorCode"`
	RequestId    int    `json:"requestId"`
}

type MediaRequest struct {
	TheMovieDbId int `json:"theMovieDbId"`
}
