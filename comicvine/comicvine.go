// Package comicvine is the library behind the comicvine CLI: the HTTP client,
// request shaping, and the typed data models for the ComicVine REST API at
// https://comicvine.gamespot.com/api/
//
// The ComicVine API is a documented JSON REST API requiring a free API key
// (COMICVINE_API_KEY). All responses use a common envelope with an "error"
// field, a "results" field, and pagination metadata. The client here wraps
// that envelope, paces requests to stay well below the 200 req/hour limit, and
// retries transient failures.
package comicvine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent is the User-Agent sent with every request.
const DefaultUserAgent = "comicvine/dev (+https://github.com/tamnd/comicvine-cli)"

// DefaultBaseURL is the root of the ComicVine REST API.
const DefaultBaseURL = "https://comicvine.gamespot.com/api"

// ErrNoAPIKey is returned when COMICVINE_API_KEY is not set.
var ErrNoAPIKey = errors.New("COMICVINE_API_KEY is not set; get a free key at https://comicvine.gamespot.com/api/")

// ErrNotFound is returned when the API returns 0 results for a detail lookup.
var ErrNotFound = errors.New("not found")

// ErrRateLimited is returned on HTTP 429.
var ErrRateLimited = errors.New("rate limited")

// Config holds constructor parameters for a Client.
type Config struct {
	BaseURL   string
	UserAgent string
	APIKey    string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults. Rate is 20s (3 req/min, 180/hr),
// safely below the 200 req/hr limit.
func DefaultConfig() Config {
	return Config{
		BaseURL:   DefaultBaseURL,
		UserAgent: DefaultUserAgent,
		Rate:      20 * time.Second,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the ComicVine REST API.
type Client struct {
	baseURL   string
	apiKey    string
	http      *http.Client
	userAgent string
	rate      time.Duration
	retries   int
	mu        sync.Mutex
	last      time.Time
}

// NewClient returns a Client built from cfg.
func NewClient(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, ErrNoAPIKey
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:    cfg.APIKey,
		http:      &http.Client{Timeout: cfg.Timeout},
		userAgent: cfg.UserAgent,
		rate:      cfg.Rate,
		retries:   cfg.Retries,
	}, nil
}

// ---- envelope ---------------------------------------------------------------

// envelope is the common API response wrapper.
type envelope struct {
	Error        string          `json:"error"`
	StatusCode   int             `json:"status_code"`
	NumberOfTotal int            `json:"number_of_total_results"`
	NumberOfPage int             `json:"number_of_page_results"`
	Limit        int             `json:"limit"`
	Offset       int             `json:"offset"`
	Results      json.RawMessage `json:"results"`
}

// ---- Search -----------------------------------------------------------------

// SearchResult is a unified result from the /search endpoint.
type SearchResult struct {
	ID           int    `json:"id"`
	ResourceType string `json:"resource_type"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ImageURL     string `json:"image_url"`
	APIURL       string `json:"api_url"`
	SiteURL      string `json:"site_url"`
}

type rawSearchResult struct {
	ID           int    `json:"id"`
	ResourceType string `json:"resource_type"`
	Name         string `json:"name"`
	Deck         string `json:"deck"`
	Image        struct {
		OriginalURL string `json:"original_url"`
	} `json:"image"`
	APIDetailURL  string `json:"api_detail_url"`
	SiteDetailURL string `json:"site_detail_url"`
}

// Search queries the ComicVine search endpoint.
func (c *Client) Search(ctx context.Context, query, resourceType string, limit int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("format", "json")
	params.Set("field_list", "id,resource_type,name,deck,image,api_detail_url,site_detail_url")
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	} else {
		params.Set("limit", "10")
	}
	if resourceType != "" {
		params.Set("resources", resourceType)
	}

	body, err := c.get(ctx, c.baseURL+"/search/", params)
	if err != nil {
		return nil, err
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}
	if env.Error != "OK" {
		return nil, fmt.Errorf("api error: %s", env.Error)
	}

	var raw []rawSearchResult
	if err := json.Unmarshal(env.Results, &raw); err != nil {
		return nil, fmt.Errorf("parse search results: %w", err)
	}

	out := make([]SearchResult, 0, len(raw))
	for _, r := range raw {
		out = append(out, SearchResult{
			ID:           r.ID,
			ResourceType: r.ResourceType,
			Name:         r.Name,
			Description:  r.Deck,
			ImageURL:     r.Image.OriginalURL,
			APIURL:       r.APIDetailURL,
			SiteURL:      r.SiteDetailURL,
		})
	}
	return out, nil
}

// ---- Issue ------------------------------------------------------------------

// Issue is a single comic issue record.
type Issue struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	IssueNumber string   `json:"issue_number"`
	Volume      string   `json:"volume"`
	VolumeID    int      `json:"volume_id"`
	CoverDate   string   `json:"cover_date"`
	StoreDate   string   `json:"store_date"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image_url"`
	Creators    []string `json:"creators"`
	Characters  []string `json:"characters"`
	StoryArcs   []string `json:"story_arcs"`
	SiteURL     string   `json:"site_url"`
}

type rawIssue struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	IssueNumber string `json:"issue_number"`
	Volume      struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"volume"`
	CoverDate string `json:"cover_date"`
	StoreDate string `json:"store_date"`
	Deck      string `json:"deck"`
	Image     struct {
		OriginalURL string `json:"original_url"`
	} `json:"image"`
	PersonCredits []struct {
		Name string `json:"name"`
	} `json:"person_credits"`
	CharacterCredits []struct {
		Name string `json:"name"`
	} `json:"character_credits"`
	StoryArcCredits []struct {
		Name string `json:"name"`
	} `json:"story_arc_credits"`
	SiteDetailURL string `json:"site_detail_url"`
}

// Issue fetches a single issue by its ComicVine ID.
func (c *Client) Issue(ctx context.Context, id int) (*Issue, error) {
	params := url.Values{}
	params.Set("format", "json")
	params.Set("field_list", "id,name,issue_number,volume,cover_date,store_date,deck,image,person_credits,character_credits,story_arc_credits,site_detail_url")

	body, err := c.get(ctx, fmt.Sprintf("%s/issue/4000-%d/", c.baseURL, id), params)
	if err != nil {
		return nil, err
	}

	var env struct {
		Error   string   `json:"error"`
		Results rawIssue `json:"results"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse issue response: %w", err)
	}
	if env.Error != "OK" {
		return nil, fmt.Errorf("api error: %s", env.Error)
	}
	if env.Results.ID == 0 {
		return nil, ErrNotFound
	}

	r := env.Results
	iss := &Issue{
		ID:          r.ID,
		Name:        r.Name,
		IssueNumber: r.IssueNumber,
		Volume:      r.Volume.Name,
		VolumeID:    r.Volume.ID,
		CoverDate:   r.CoverDate,
		StoreDate:   r.StoreDate,
		Description: r.Deck,
		ImageURL:    r.Image.OriginalURL,
		SiteURL:     r.SiteDetailURL,
	}
	for _, p := range r.PersonCredits {
		iss.Creators = append(iss.Creators, p.Name)
	}
	for _, ch := range r.CharacterCredits {
		iss.Characters = append(iss.Characters, ch.Name)
	}
	for _, sa := range r.StoryArcCredits {
		iss.StoryArcs = append(iss.StoryArcs, sa.Name)
	}
	return iss, nil
}

// ---- Volume -----------------------------------------------------------------

// Volume is a comic series/volume record.
type Volume struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Publisher   string `json:"publisher"`
	PublisherID int    `json:"publisher_id"`
	StartYear   int    `json:"start_year"`
	IssueCount  int    `json:"issue_count"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	SiteURL     string `json:"site_url"`
}

type rawVolume struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Publisher struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"publisher"`
	StartYear  string `json:"start_year"`
	CountOfIssues int `json:"count_of_issues"`
	Deck       string `json:"deck"`
	Image      struct {
		OriginalURL string `json:"original_url"`
	} `json:"image"`
	SiteDetailURL string `json:"site_detail_url"`
}

// Volume fetches a single volume by its ComicVine ID.
func (c *Client) Volume(ctx context.Context, id int) (*Volume, error) {
	params := url.Values{}
	params.Set("format", "json")
	params.Set("field_list", "id,name,publisher,start_year,count_of_issues,deck,image,site_detail_url")

	body, err := c.get(ctx, fmt.Sprintf("%s/volume/4050-%d/", c.baseURL, id), params)
	if err != nil {
		return nil, err
	}

	var env struct {
		Error   string    `json:"error"`
		Results rawVolume `json:"results"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse volume response: %w", err)
	}
	if env.Error != "OK" {
		return nil, fmt.Errorf("api error: %s", env.Error)
	}
	if env.Results.ID == 0 {
		return nil, ErrNotFound
	}

	r := env.Results
	startYear, _ := strconv.Atoi(r.StartYear)
	return &Volume{
		ID:          r.ID,
		Name:        r.Name,
		Publisher:   r.Publisher.Name,
		PublisherID: r.Publisher.ID,
		StartYear:   startYear,
		IssueCount:  r.CountOfIssues,
		Description: r.Deck,
		ImageURL:    r.Image.OriginalURL,
		SiteURL:     r.SiteDetailURL,
	}, nil
}

// ---- Character --------------------------------------------------------------

// Character is a comic character record.
type Character struct {
	ID                int      `json:"id"`
	Name              string   `json:"name"`
	RealName          string   `json:"real_name"`
	Aliases           []string `json:"aliases"`
	Publisher         string   `json:"publisher"`
	PublisherID       int      `json:"publisher_id"`
	FirstAppearance   string   `json:"first_appearance"`
	FirstAppearanceID int      `json:"first_appearance_id"`
	Description       string   `json:"description"`
	ImageURL          string   `json:"image_url"`
	SiteURL           string   `json:"site_url"`
}

type rawCharacter struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	Aliases  string `json:"aliases"`
	Publisher struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"publisher"`
	FirstAppearance struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"first_appeared_in_issue"`
	Deck  string `json:"deck"`
	Image struct {
		OriginalURL string `json:"original_url"`
	} `json:"image"`
	SiteDetailURL string `json:"site_detail_url"`
}

// Character fetches a character by its ComicVine ID.
func (c *Client) Character(ctx context.Context, id int) (*Character, error) {
	params := url.Values{}
	params.Set("format", "json")
	params.Set("field_list", "id,name,real_name,aliases,publisher,first_appeared_in_issue,deck,image,site_detail_url")

	body, err := c.get(ctx, fmt.Sprintf("%s/character/4005-%d/", c.baseURL, id), params)
	if err != nil {
		return nil, err
	}

	var env struct {
		Error   string       `json:"error"`
		Results rawCharacter `json:"results"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse character response: %w", err)
	}
	if env.Error != "OK" {
		return nil, fmt.Errorf("api error: %s", env.Error)
	}
	if env.Results.ID == 0 {
		return nil, ErrNotFound
	}

	return rawCharacterToRecord(env.Results), nil
}

// CharacterByName looks up a character by name, returning the first match.
func (c *Client) CharacterByName(ctx context.Context, name string) (*Character, error) {
	params := url.Values{}
	params.Set("format", "json")
	params.Set("filter", "name:"+name)
	params.Set("field_list", "id,name,real_name,aliases,publisher,first_appeared_in_issue,deck,image,site_detail_url")
	params.Set("limit", "1")

	body, err := c.get(ctx, c.baseURL+"/characters/", params)
	if err != nil {
		return nil, err
	}

	var env struct {
		Error   string         `json:"error"`
		Results []rawCharacter `json:"results"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse characters response: %w", err)
	}
	if env.Error != "OK" {
		return nil, fmt.Errorf("api error: %s", env.Error)
	}
	if len(env.Results) == 0 {
		return nil, ErrNotFound
	}
	return rawCharacterToRecord(env.Results[0]), nil
}

func rawCharacterToRecord(r rawCharacter) *Character {
	aliases := []string{}
	if r.Aliases != "" {
		for _, a := range strings.Split(r.Aliases, "\n") {
			if t := strings.TrimSpace(a); t != "" {
				aliases = append(aliases, t)
			}
		}
	}
	ch := &Character{
		ID:                r.ID,
		Name:              r.Name,
		RealName:          r.RealName,
		Aliases:           aliases,
		Publisher:         r.Publisher.Name,
		PublisherID:       r.Publisher.ID,
		FirstAppearance:   r.FirstAppearance.Name,
		FirstAppearanceID: r.FirstAppearance.ID,
		Description:       r.Deck,
		ImageURL:          r.Image.OriginalURL,
		SiteURL:           r.SiteDetailURL,
	}
	return ch
}

// ---- Publisher --------------------------------------------------------------

// Publisher is a comic publisher record.
type Publisher struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Aliases     string `json:"aliases"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	SiteURL     string `json:"site_url"`
}

type rawPublisher struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Aliases string `json:"aliases"`
	Deck    string `json:"deck"`
	Image   struct {
		OriginalURL string `json:"original_url"`
	} `json:"image"`
	SiteDetailURL string `json:"site_detail_url"`
}

// Publisher fetches a publisher by its ComicVine ID.
func (c *Client) Publisher(ctx context.Context, id int) (*Publisher, error) {
	params := url.Values{}
	params.Set("format", "json")
	params.Set("field_list", "id,name,aliases,deck,image,site_detail_url")

	body, err := c.get(ctx, fmt.Sprintf("%s/publisher/4010-%d/", c.baseURL, id), params)
	if err != nil {
		return nil, err
	}

	var env struct {
		Error   string       `json:"error"`
		Results rawPublisher `json:"results"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse publisher response: %w", err)
	}
	if env.Error != "OK" {
		return nil, fmt.Errorf("api error: %s", env.Error)
	}
	if env.Results.ID == 0 {
		return nil, ErrNotFound
	}

	r := env.Results
	return &Publisher{
		ID:          r.ID,
		Name:        r.Name,
		Aliases:     r.Aliases,
		Description: r.Deck,
		ImageURL:    r.Image.OriginalURL,
		SiteURL:     r.SiteDetailURL,
	}, nil
}

// ---- Person (Creator) -------------------------------------------------------

// Person is a comic creator record.
type Person struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	RealName    string `json:"real_name"`
	Aliases     string `json:"aliases"`
	BirthDate   string `json:"birth_date"`
	Hometown    string `json:"hometown"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	SiteURL     string `json:"site_url"`
}

type rawPerson struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	RealName  string `json:"real_name"`
	Aliases   string `json:"aliases"`
	BirthDate string `json:"birth"`
	HomeTown  string `json:"hometown"`
	Deck      string `json:"deck"`
	Image     struct {
		OriginalURL string `json:"original_url"`
	} `json:"image"`
	SiteDetailURL string `json:"site_detail_url"`
}

// Person fetches a creator by their ComicVine ID.
func (c *Client) Person(ctx context.Context, id int) (*Person, error) {
	params := url.Values{}
	params.Set("format", "json")
	params.Set("field_list", "id,name,real_name,aliases,birth,hometown,deck,image,site_detail_url")

	body, err := c.get(ctx, fmt.Sprintf("%s/person/4040-%d/", c.baseURL, id), params)
	if err != nil {
		return nil, err
	}

	var env struct {
		Error   string    `json:"error"`
		Results rawPerson `json:"results"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse person response: %w", err)
	}
	if env.Error != "OK" {
		return nil, fmt.Errorf("api error: %s", env.Error)
	}
	if env.Results.ID == 0 {
		return nil, ErrNotFound
	}

	r := env.Results
	return &Person{
		ID:          r.ID,
		Name:        r.Name,
		RealName:    r.RealName,
		Aliases:     r.Aliases,
		BirthDate:   r.BirthDate,
		Hometown:    r.HomeTown,
		Description: r.Deck,
		ImageURL:    r.Image.OriginalURL,
		SiteURL:     r.SiteDetailURL,
	}, nil
}

// PersonByName looks up a creator by name, returning the first match.
func (c *Client) PersonByName(ctx context.Context, name string) (*Person, error) {
	params := url.Values{}
	params.Set("format", "json")
	params.Set("filter", "name:"+name)
	params.Set("field_list", "id,name,real_name,aliases,birth,hometown,deck,image,site_detail_url")
	params.Set("limit", "1")

	body, err := c.get(ctx, c.baseURL+"/people/", params)
	if err != nil {
		return nil, err
	}

	var env struct {
		Error   string      `json:"error"`
		Results []rawPerson `json:"results"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse people response: %w", err)
	}
	if env.Error != "OK" {
		return nil, fmt.Errorf("api error: %s", env.Error)
	}
	if len(env.Results) == 0 {
		return nil, ErrNotFound
	}

	r := env.Results[0]
	return &Person{
		ID:          r.ID,
		Name:        r.Name,
		RealName:    r.RealName,
		Aliases:     r.Aliases,
		BirthDate:   r.BirthDate,
		Hometown:    r.HomeTown,
		Description: r.Deck,
		ImageURL:    r.Image.OriginalURL,
		SiteURL:     r.SiteDetailURL,
	}, nil
}

// ---- IssueStub (for issues list) -------------------------------------------

// IssueStub is a summary record for listing issues in a volume.
type IssueStub struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	IssueNumber string `json:"issue_number"`
	CoverDate   string `json:"cover_date"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	SiteURL     string `json:"site_url"`
}

type rawIssueStub struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	IssueNumber string `json:"issue_number"`
	CoverDate   string `json:"cover_date"`
	Deck        string `json:"deck"`
	Image       struct {
		OriginalURL string `json:"original_url"`
	} `json:"image"`
	SiteDetailURL string `json:"site_detail_url"`
}

// Issues lists the issues in a volume, optionally filtered by issue number range.
// from and to are inclusive; 0 means no bound. limit 0 means no limit.
func (c *Client) Issues(ctx context.Context, volumeID, from, to, limit int) ([]IssueStub, error) {
	params := url.Values{}
	params.Set("format", "json")
	params.Set("filter", fmt.Sprintf("volume:%d", volumeID))
	params.Set("field_list", "id,name,issue_number,cover_date,deck,image,site_detail_url")
	params.Set("sort", "issue_number:asc")
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	} else {
		params.Set("limit", "100")
	}

	body, err := c.get(ctx, c.baseURL+"/issues/", params)
	if err != nil {
		return nil, err
	}

	var env struct {
		Error   string         `json:"error"`
		Results []rawIssueStub `json:"results"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse issues response: %w", err)
	}
	if env.Error != "OK" {
		return nil, fmt.Errorf("api error: %s", env.Error)
	}

	var out []IssueStub
	for _, r := range env.Results {
		if from > 0 || to > 0 {
			n, _ := strconv.Atoi(r.IssueNumber)
			if from > 0 && n < from {
				continue
			}
			if to > 0 && n > to {
				continue
			}
		}
		out = append(out, IssueStub{
			ID:          r.ID,
			Name:        r.Name,
			IssueNumber: r.IssueNumber,
			CoverDate:   r.CoverDate,
			Description: r.Deck,
			ImageURL:    r.Image.OriginalURL,
			SiteURL:     r.SiteDetailURL,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// ---- HTTP -------------------------------------------------------------------

func (c *Client) get(ctx context.Context, rawURL string, params url.Values) ([]byte, error) {
	params.Set("api_key", c.apiKey)

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	u.RawQuery = params.Encode()
	full := u.String()

	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, full)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, true, ErrRateLimited
	}
	if resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	if c.rate <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
