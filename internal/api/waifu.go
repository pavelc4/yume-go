package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yume-go/internal/config"
)

type Waifu struct {
	URL       string   `json:"url"`
	ImageID   string   `json:"image_id"`
	Name      string   `json:"name"`
	Tags      []string `json:"tags"`
	Source    string   `json:"source"`
	Character string   `json:"character"`
	Origin    string   `json:"origin"`
	Artist    string   `json:"artist"`
	PageURL   string   `json:"page_url"`
}

type APIClient struct {
	WaifuImURL   string
	WaifuPicsURL string
	WaifuItURL   string

	http *http.Client

	failCount map[string]int
	lastFail  map[string]time.Time
}

func NewAPIClient(waifuImURL, waifuPicsURL, waifuItURL string) *APIClient {
	return &APIClient{
		WaifuImURL:   waifuImURL,
		WaifuPicsURL: waifuPicsURL,
		WaifuItURL:   waifuItURL,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		failCount: make(map[string]int),
		lastFail:  make(map[string]time.Time),
	}
}

func (c *APIClient) resetFail(name string) { c.failCount[name] = 0 }
func (c *APIClient) markFail(name string)  { c.failCount[name]++; c.lastFail[name] = time.Now() }

func (c *APIClient) isTemporarilyUnhealthy(name string) bool {
	if c.failCount[name] >= 3 && time.Since(c.lastFail[name]) < 2*time.Minute {
		return time.Since(c.lastFail[name]) < 5*time.Minute
	}
	return false
}
func deriveIDFromURL(s string) string {
	if s == "" {
		return ""
	}
	last := s
	if i := strings.LastIndex(s, "/"); i != -1 {
		last = s[i+1:]
	}
	if j := strings.Index(last, "?"); j != -1 {
		last = last[:j]
	}
	if k := strings.LastIndex(last, "."); k != -1 {
		last = last[:k]
	}
	return strings.TrimSpace(last)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func StableNumericID(prefix, raw string) string {
	base := prefix + ":" + raw
	h := fnv.New64a()
	_, _ = h.Write([]byte(base))
	return strconv.FormatUint(h.Sum64(), 10)
}

func parseWeights(spec string) map[string]int {
	out := map[string]int{}
	parts := strings.Split(spec, ",")
	for _, p := range parts {
		kv := strings.Split(strings.TrimSpace(p), ":")
		if len(kv) != 2 {
			continue
		}
		name := strings.TrimSpace(kv[0])
		w, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil || w <= 0 {
			continue
		}
		out[name] = w
	}
	return out
}

func pickWeighted(weights map[string]int, unhealthy func(name string) bool) string {
	total := 0
	effective := map[string]int{}
	for name, w := range weights {
		if unhealthy != nil && unhealthy(name) {
			w = 1
		}
		effective[name] = w
		total += w
	}
	if total <= 0 {
		return ""
	}
	nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(total)))
	n := int(nBig.Int64())
	cum := 0
	for name, w := range effective {
		cum += w
		if n < cum {
			return name
		}
	}
	return ""
}

func (c *APIClient) FetchRandomWaifu(isNSFW bool, apiPriority []string, cfg *config.Config) (*Waifu, error) {
	weights := parseWeights(cfg.WaifuWeights)
	chosen := pickWeighted(weights, c.isTemporarilyUnhealthy)

	tryOrder := make([]string, 0, 4)
	if chosen != "" {
		tryOrder = append(tryOrder, chosen)
	}
	exists := map[string]bool{}
	if chosen != "" {
		exists[chosen] = true
	}
	for _, s := range apiPriority {
		if s == "" || exists[s] {
			continue
		}
		tryOrder = append(tryOrder, s)
		exists[s] = true
	}
	if len(tryOrder) == 0 {
		return nil, errors.New("no sources to try")
	}

	var lastErr error
	for _, src := range tryOrder {
		var w *Waifu
		var err error

		switch src {
		case "waifu.im":
			w, err = c.fetchFromWaifuIm(isNSFW)
		case "waifu.pics":
			w, err = c.fetchFromWaifuPics(isNSFW)
		case "waifu.it":
			w, err = c.fetchFromWaifuIt(isNSFW)
		default:
			err = fmt.Errorf("unknown source: %s", src)
		}

		if err != nil || w == nil || w.URL == "" {
			c.markFail(src)
			lastErr = fmt.Errorf("source %s failed: %w", src, err)
			log.Printf("[gacha] source=%s error=%v", src, err)
			continue
		}

		c.resetFail(src)
		w.Source = src
		return w, nil
	}
	return nil, lastErr
}

func (c *APIClient) fetchFromWaifuIm(isNSFW bool) (*Waifu, error) {
	u, _ := url.Parse(c.WaifuImURL)
	q := u.Query()
	q.Set("is_nsfw", strconv.FormatBool(isNSFW))
	q.Set("many", "false")
	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	req.Header.Set("User-Agent", "yume-go/1.0")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("waifu.im bad status: %d", resp.StatusCode)
	}

	var payload struct {
		Images []struct {
			URL       string   `json:"url"`
			ImageID   string   `json:"image_id"`
			Tags      []string `json:"tags"`
			Source    string   `json:"source"`
			Character string   `json:"character"`
			Origin    string   `json:"origin"`
			Artist    string   `json:"artist"`
			PageURL   string   `json:"page_url"`
			Name      string   `json:"name"`
		} `json:"images"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Images) == 0 {
		return nil, errors.New("waifu.im empty result")
	}
	img := payload.Images[0]

	num := StableNumericID("im", img.ImageID)
	return &Waifu{
		URL:       img.URL,
		ImageID:   num,
		Name:      img.Name,
		Tags:      img.Tags,
		Character: img.Character,
		Origin:    img.Origin,
		Artist:    img.Artist,
		PageURL:   img.PageURL,
	}, nil
}

func (c *APIClient) fetchFromWaifuPics(isNSFW bool) (*Waifu, error) {
	mode := "sfw"
	if isNSFW {
		mode = "nsfw"
	}
	u := fmt.Sprintf("%s/%s/waifu", strings.TrimRight(c.WaifuPicsURL, "/"), mode)

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("User-Agent", "yume-go/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("waifu.pics bad status: %d", resp.StatusCode)
	}

	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.URL == "" {
		return nil, errors.New("waifu.pics empty url")
	}

	raw := deriveIDFromURL(payload.URL)
	num := StableNumericID("pics", raw)
	return &Waifu{
		URL:     payload.URL,
		ImageID: num,
	}, nil
}

func (c *APIClient) fetchFromWaifuIt(isNSFW bool) (*Waifu, error) {
	u := fmt.Sprintf("%s/random?nsfw=%t", strings.TrimRight(c.WaifuItURL, "/"), isNSFW)

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("User-Agent", "yume-go/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("waifu.it bad status: %d", resp.StatusCode)
	}

	var payload struct {
		URL       string   `json:"url"`
		ID        string   `json:"id"`
		Name      string   `json:"name"`
		Tags      []string `json:"tags"`
		Character string   `json:"character"`
		Origin    string   `json:"origin"`
		Artist    string   `json:"artist"`
		PageURL   string   `json:"page_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.URL == "" {
		return nil, errors.New("waifu.it empty url")
	}

	raw := firstNonEmpty(payload.ID, deriveIDFromURL(payload.URL))
	num := StableNumericID("it", raw)
	return &Waifu{
		URL:       payload.URL,
		ImageID:   num,
		Name:      payload.Name,
		Tags:      payload.Tags,
		Character: payload.Character,
		Origin:    payload.Origin,
		Artist:    payload.Artist,
		PageURL:   payload.PageURL,
	}, nil
}
