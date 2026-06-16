package comicvine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/comicvine-cli/comicvine"
)

func writeEnvelope(w http.ResponseWriter, results interface{}) {
	type envelope struct {
		Error        string      `json:"error"`
		StatusCode   int         `json:"status_code"`
		NumberOfTotal int        `json:"number_of_total_results"`
		NumberOfPage int         `json:"number_of_page_results"`
		Results      interface{} `json:"results"`
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(envelope{
		Error:        "OK",
		StatusCode:   1,
		NumberOfTotal: 1,
		NumberOfPage: 1,
		Results:      results,
	})
}

func newTestServer(t *testing.T) (*httptest.Server, *comicvine.Client) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request missing User-Agent")
		}
		if r.URL.Query().Get("api_key") == "" {
			t.Error("request missing api_key")
		}
		switch r.URL.Path {
		case "/search/":
			writeEnvelope(w, []map[string]interface{}{
				{
					"id":            1,
					"resource_type": "character",
					"name":          "Batman",
					"deck":          "The Dark Knight",
					"image":         map[string]string{"original_url": "https://example.com/batman.jpg"},
					"api_detail_url":  "https://comicvine.gamespot.com/api/character/4005-1/",
					"site_detail_url": "https://comicvine.gamespot.com/batman/4005-1/",
				},
			})
		case "/issue/4000-100/":
			writeEnvelope(w, map[string]interface{}{
				"id":           100,
				"name":         "The Dark Knight Returns",
				"issue_number": "1",
				"volume": map[string]interface{}{
					"id":   796,
					"name": "The Dark Knight Returns",
				},
				"cover_date": "1986-03",
				"store_date": "1986-02-19",
				"deck":       "First issue of the miniseries",
				"image":      map[string]string{"original_url": "https://example.com/dkr1.jpg"},
				"person_credits":    []map[string]string{{"name": "Frank Miller"}},
				"character_credits": []map[string]string{{"name": "Batman"}},
				"story_arc_credits": []map[string]string{},
				"site_detail_url":   "https://comicvine.gamespot.com/the-dark-knight-returns-1/4000-100/",
			})
		case "/volume/4050-796/":
			writeEnvelope(w, map[string]interface{}{
				"id":   796,
				"name": "The Dark Knight Returns",
				"publisher": map[string]interface{}{
					"id":   10,
					"name": "DC Comics",
				},
				"start_year":      "1986",
				"count_of_issues": 4,
				"deck":            "Frank Miller's landmark miniseries",
				"image":           map[string]string{"original_url": "https://example.com/dkr.jpg"},
				"site_detail_url": "https://comicvine.gamespot.com/the-dark-knight-returns/4050-796/",
			})
		case "/character/4005-1490/":
			writeEnvelope(w, map[string]interface{}{
				"id":       1490,
				"name":     "Batman",
				"real_name": "Bruce Wayne",
				"aliases":  "The Dark Knight\nCaped Crusader",
				"publisher": map[string]interface{}{
					"id":   10,
					"name": "DC Comics",
				},
				"first_appeared_in_issue": map[string]interface{}{
					"id":   8194,
					"name": "Detective Comics #27",
				},
				"deck":            "The Dark Knight of Gotham City",
				"image":           map[string]string{"original_url": "https://example.com/batman.jpg"},
				"site_detail_url": "https://comicvine.gamespot.com/batman/4005-1490/",
			})
		case "/characters/":
			writeEnvelope(w, []map[string]interface{}{
				{
					"id":       1490,
					"name":     "Batman",
					"real_name": "Bruce Wayne",
					"aliases":  "The Dark Knight",
					"publisher": map[string]interface{}{
						"id":   10,
						"name": "DC Comics",
					},
					"first_appeared_in_issue": map[string]interface{}{
						"id":   8194,
						"name": "Detective Comics #27",
					},
					"deck":            "The Dark Knight",
					"image":           map[string]string{"original_url": "https://example.com/batman.jpg"},
					"site_detail_url": "https://comicvine.gamespot.com/batman/4005-1490/",
				},
			})
		case "/publisher/4010-10/":
			writeEnvelope(w, map[string]interface{}{
				"id":              10,
				"name":            "DC Comics",
				"aliases":         "Detective Comics",
				"deck":            "The home of Batman and Superman",
				"image":           map[string]string{"original_url": "https://example.com/dc.jpg"},
				"site_detail_url": "https://comicvine.gamespot.com/dc-comics/4010-10/",
			})
		case "/person/4040-1457/":
			writeEnvelope(w, map[string]interface{}{
				"id":              1457,
				"name":            "Stan Lee",
				"real_name":       "Stanley Martin Lieber",
				"aliases":         "",
				"birth":           "1922-12-28",
				"hometown":        "New York, New York",
				"deck":            "Co-creator of many Marvel characters",
				"image":           map[string]string{"original_url": "https://example.com/stan.jpg"},
				"site_detail_url": "https://comicvine.gamespot.com/stan-lee/4040-1457/",
			})
		case "/people/":
			writeEnvelope(w, []map[string]interface{}{
				{
					"id":              1457,
					"name":            "Stan Lee",
					"real_name":       "Stanley Martin Lieber",
					"aliases":         "",
					"birth":           "1922-12-28",
					"hometown":        "New York, New York",
					"deck":            "Co-creator of many Marvel characters",
					"image":           map[string]string{"original_url": "https://example.com/stan.jpg"},
					"site_detail_url": "https://comicvine.gamespot.com/stan-lee/4040-1457/",
				},
			})
		case "/issues/":
			writeEnvelope(w, []map[string]interface{}{
				{
					"id":              100,
					"name":            "The Dark Knight Returns",
					"issue_number":    "1",
					"cover_date":      "1986-03",
					"deck":            "First issue",
					"image":           map[string]string{"original_url": "https://example.com/1.jpg"},
					"site_detail_url": "https://comicvine.gamespot.com/the-dark-knight-returns-1/4000-100/",
				},
				{
					"id":              101,
					"name":            "The Dark Knight Triumphant",
					"issue_number":    "2",
					"cover_date":      "1986-06",
					"deck":            "Second issue",
					"image":           map[string]string{"original_url": "https://example.com/2.jpg"},
					"site_detail_url": "https://comicvine.gamespot.com/the-dark-knight-triumphant/4000-101/",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	cfg := comicvine.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.APIKey = "test-key"
	cfg.Rate = 0
	cfg.Retries = 0
	client, err := comicvine.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return srv, client
}

func TestSearch(t *testing.T) {
	_, client := newTestServer(t)

	results, err := client.Search(context.Background(), "batman", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.Name != "Batman" {
		t.Errorf("Name = %q, want Batman", r.Name)
	}
	if r.ResourceType != "character" {
		t.Errorf("ResourceType = %q, want character", r.ResourceType)
	}
	if r.Description == "" {
		t.Error("Description is empty")
	}
	if r.SiteURL == "" {
		t.Error("SiteURL is empty")
	}
}

func TestIssue(t *testing.T) {
	_, client := newTestServer(t)

	iss, err := client.Issue(context.Background(), 100)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if iss.ID != 100 {
		t.Errorf("ID = %d, want 100", iss.ID)
	}
	if iss.IssueNumber != "1" {
		t.Errorf("IssueNumber = %q, want 1", iss.IssueNumber)
	}
	if iss.Volume != "The Dark Knight Returns" {
		t.Errorf("Volume = %q", iss.Volume)
	}
	if len(iss.Creators) == 0 {
		t.Error("Creators is empty")
	}
	if iss.ImageURL == "" {
		t.Error("ImageURL is empty")
	}
}

func TestVolume(t *testing.T) {
	_, client := newTestServer(t)

	vol, err := client.Volume(context.Background(), 796)
	if err != nil {
		t.Fatalf("Volume: %v", err)
	}
	if vol.ID != 796 {
		t.Errorf("ID = %d, want 796", vol.ID)
	}
	if vol.Publisher != "DC Comics" {
		t.Errorf("Publisher = %q, want DC Comics", vol.Publisher)
	}
	if vol.StartYear != 1986 {
		t.Errorf("StartYear = %d, want 1986", vol.StartYear)
	}
	if vol.IssueCount != 4 {
		t.Errorf("IssueCount = %d, want 4", vol.IssueCount)
	}
}

func TestCharacterByID(t *testing.T) {
	_, client := newTestServer(t)

	ch, err := client.Character(context.Background(), 1490)
	if err != nil {
		t.Fatalf("Character: %v", err)
	}
	if ch.ID != 1490 {
		t.Errorf("ID = %d, want 1490", ch.ID)
	}
	if ch.Name != "Batman" {
		t.Errorf("Name = %q, want Batman", ch.Name)
	}
	if ch.RealName != "Bruce Wayne" {
		t.Errorf("RealName = %q, want Bruce Wayne", ch.RealName)
	}
	if len(ch.Aliases) == 0 {
		t.Error("Aliases is empty")
	}
}

func TestCharacterByName(t *testing.T) {
	_, client := newTestServer(t)

	ch, err := client.CharacterByName(context.Background(), "Batman")
	if err != nil {
		t.Fatalf("CharacterByName: %v", err)
	}
	if ch.Name != "Batman" {
		t.Errorf("Name = %q, want Batman", ch.Name)
	}
}

func TestPublisher(t *testing.T) {
	_, client := newTestServer(t)

	pub, err := client.Publisher(context.Background(), 10)
	if err != nil {
		t.Fatalf("Publisher: %v", err)
	}
	if pub.ID != 10 {
		t.Errorf("ID = %d, want 10", pub.ID)
	}
	if pub.Name != "DC Comics" {
		t.Errorf("Name = %q, want DC Comics", pub.Name)
	}
}

func TestPersonByID(t *testing.T) {
	_, client := newTestServer(t)

	p, err := client.Person(context.Background(), 1457)
	if err != nil {
		t.Fatalf("Person: %v", err)
	}
	if p.ID != 1457 {
		t.Errorf("ID = %d, want 1457", p.ID)
	}
	if p.Name != "Stan Lee" {
		t.Errorf("Name = %q, want Stan Lee", p.Name)
	}
	if p.BirthDate == "" {
		t.Error("BirthDate is empty")
	}
}

func TestPersonByName(t *testing.T) {
	_, client := newTestServer(t)

	p, err := client.PersonByName(context.Background(), "Stan Lee")
	if err != nil {
		t.Fatalf("PersonByName: %v", err)
	}
	if p.Name != "Stan Lee" {
		t.Errorf("Name = %q, want Stan Lee", p.Name)
	}
}

func TestIssues(t *testing.T) {
	_, client := newTestServer(t)

	issues, err := client.Issues(context.Background(), 796, 0, 0, 0)
	if err != nil {
		t.Fatalf("Issues: %v", err)
	}
	if len(issues) != 2 {
		t.Errorf("got %d issues, want 2", len(issues))
	}
	if issues[0].IssueNumber != "1" {
		t.Errorf("first issue number = %q, want 1", issues[0].IssueNumber)
	}
}

func TestNoAPIKey(t *testing.T) {
	cfg := comicvine.DefaultConfig()
	cfg.APIKey = ""
	_, err := comicvine.NewClient(cfg)
	if err == nil {
		t.Fatal("expected error for missing API key, got nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := comicvine.DefaultConfig()
	if cfg.BaseURL == "" {
		t.Error("DefaultConfig has empty BaseURL")
	}
	if cfg.UserAgent == "" {
		t.Error("DefaultConfig has empty UserAgent")
	}
	if cfg.Retries <= 0 {
		t.Error("DefaultConfig has non-positive Retries")
	}
	if cfg.Rate <= 0 {
		t.Error("DefaultConfig has non-positive Rate")
	}
}
