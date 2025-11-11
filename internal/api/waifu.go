package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type Waifu struct {
	URL     string   `json:"url"`
	ImageID string   `json:"image_id"`
	Name    string   `json:"name"`
	Tags    []string `json:"tags"`
	Source  string   `json:"source"`
}

type APIClient struct {
	WaifuImURL   string
	WaifuPicsURL string
	WaifuItURL   string

	fetchers map[string]func(bool) (*Waifu, error)
}

func NewAPIClient(waifuImURL, waifuPicsURL, waifuItURL string) *APIClient {
	client := &APIClient{
		WaifuImURL:   waifuImURL,
		WaifuPicsURL: waifuPicsURL,
		WaifuItURL:   waifuItURL,
	}

	client.fetchers = map[string]func(bool) (*Waifu, error){
		"waifu.im":   client.fetchFromWaifuIm,
		"waifu.pics": client.fetchFromWaifuPics,
		"waifu.it":   client.fetchFromWaifuIt,
	}

	return client
}

func (c *APIClient) FetchRandomWaifu(isNSFW bool, apiPriority []string) (*Waifu, error) {
	var lastErr error

	for _, apiName := range apiPriority {
		fetcher, ok := c.fetchers[apiName]
		if !ok {
			continue
		}
		waifu, err := fetcher(isNSFW)
		if err == nil && waifu != nil {
			waifu.Source = apiName
			return waifu, nil
		}
		lastErr = err
		log.Printf("Failed to fetch from %s: %v", apiName, err)
	}
	return nil, fmt.Errorf("all APIs failed, last error: %w", lastErr)
}

func (c *APIClient) fetchFromWaifuIm(isNSFW bool) (*Waifu, error) {
	params := url.Values{}
	params.Add("is_nsfw", fmt.Sprintf("%t", isNSFW))
	fullURL := fmt.Sprintf("%s?%s", c.WaifuImURL, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var result struct {
		Images []struct {
			URL     string `json:"url"`
			ImageID int    `json:"image_id"`
			Tags    []struct {
				Name string `json:"name"`
			} `json:"tags"`
		} `json:"images"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("no images found")
	}

	img := result.Images[0]
	tags := make([]string, len(img.Tags))
	for i, tag := range img.Tags {
		tags[i] = tag.Name
	}

	name := "Waifu"
	if len(tags) > 0 {
		name = tags[0]
	}

	return &Waifu{
		URL:     img.URL,
		ImageID: fmt.Sprintf("%d", img.ImageID),
		Name:    name,
		Tags:    tags,
	}, nil
}

func (c *APIClient) fetchFromWaifuPics(isNSFW bool) (*Waifu, error) {
	category := "sfw"
	if isNSFW {
		category = "nsfw"
	}
	fullURL := fmt.Sprintf("%s/%s/waifu", c.WaifuPicsURL, category)

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var result struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &Waifu{
		URL:     result.URL,
		ImageID: "waifu-pics",
		Name:    "Waifu",
		Tags:    []string{"waifu"},
	}, nil
}

func (c *APIClient) fetchFromWaifuIt(isNSFW bool) (*Waifu, error) {
	fullURL := fmt.Sprintf("%s/waifu", c.WaifuItURL)

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var result struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &Waifu{
		URL:     result.URL,
		ImageID: "waifu-it",
		Name:    result.Name,
		Tags:    []string{"waifu"},
	}, nil
}
