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

type OmbiClient struct {
	url string
	key string
}

func NewOmbiClient(url string, key string) *OmbiClient {
	return &OmbiClient{
		url: url,
		key: key,
	}
}

func (client *OmbiClient) PerformMultiSearch(query string) ([]MultiSearchResult, error) {
	searchRequest := &MulitSearchRequest{
		Movies:  true,
		TvShows: true,
		Music:   false,
		People:  false,
	}
	url := "api/v2/Search/multi/" + query

	resp, err := client.post(url, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("can't request %s from client %+v: %w", url, client, err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read response body: %w", err)
	}

	log.Printf("Response status: %s, response body: %s", resp.Status, string(body))

	if (resp.StatusCode < 200) || (resp.StatusCode >= 300) {
		return nil, fmt.Errorf("got non-2xx response status: %s", resp.Status)
	}

	var result []MultiSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("can't decode JSON response: %w", err)
	}

	return result, nil
}

func (client *OmbiClient) RequestMedia(result MultiSearchResult) error {
	request_body, err := create_request_body(result)
	if err != nil {
		return fmt.Errorf("can't create request body for %+v: %w", result, err)
	}

	request_path, err := get_media_request_path(result)
	if err != nil {
		return err
	}

	resp, err := client.post(request_path, request_body)
	if err != nil {
		return fmt.Errorf("can't request %s from client %+v: %w", request_path, client, err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("can't read response body: %w", err)
	}

	log.Printf("Response status: %s, response body: %s", resp.Status, string(body))

	if (resp.StatusCode < 200) || (resp.StatusCode >= 300) {
		return fmt.Errorf("got non-2xx response status: %s", resp.Status)
	}

	var response MediaRequestResult
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("can't decode JSON response: %w", err)
	}

	if response.IsError {
		return fmt.Errorf("got error response: %s", response.ErrorMessage)
	}

	return nil
}

func get_media_request_path(result MultiSearchResult) (string, error) {
	if result.MediaType == "movie" {
		return "api/v1/Request/movie", nil
	} else if result.MediaType == "tv" {
		return "api/v2/Requests/tv", nil
	} else {
		return "", fmt.Errorf("unknown media type %s", result.MediaType)
	}
}

func create_request_body(result MultiSearchResult) (any, error) {
	dbId, err := strconv.Atoi(result.Id)
	if err != nil {
		return nil, fmt.Errorf("can't convert %s to int: %w", result.Id, err)
	}

	request := &MediaRequest{
		TheMovieDbId: dbId,
	}

	return request, nil
}

func (client *OmbiClient) post(path string, body any) (*http.Response, error) {
	requestBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("can't encode JSON request %+v: %w", body, err)
	}

	url := fmt.Sprintf("%s/%s", client.url, path)

	request, err := http.NewRequest("POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("can't create POST request %s from client %+v: %w", url, client, err)
	}
	request.Header.Set("Content-Type", "application/json")

	log.Printf("POST %s with body %s", url, string(requestBody))

	return client.doRequest(request)
}

func (client *OmbiClient) doRequest(req *http.Request) (*http.Response, error) {
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
