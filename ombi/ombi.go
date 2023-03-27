package ombi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
		return nil, fmt.Errorf("Can't request %s from client %+v: %w", url, client, err)
	}
	defer resp.Body.Close()

	if (resp.StatusCode < 200) || (resp.StatusCode >= 300) {
		return nil, fmt.Errorf("Got non-2xx response status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Can't read response body: %w", err)
	}

	log.Printf("Response status: %s, response body: %s", resp.Status, string(body))

	var result []MultiSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("Can't decode JSON response: %w", err)
	}

	return result, nil
}

func (client *OmbiClient) post(path string, body any) (*http.Response, error) {
	requestBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("Can't encode JSON request %+v: %w", body, err)
	}

	url := fmt.Sprintf("%s/%s", client.url, path)

	request, err := http.NewRequest("POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("Can't create POST request %s from client %+v: %w", url, client, err)
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
