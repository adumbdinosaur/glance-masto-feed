package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"io"
	"encoding/json"
)

type MastodonStatus struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Account   struct {
		DisplayName string `json:"display_name"`
		Acct        string `json:"acct"`
	} `json:"account"`
	URL string `json:"url"`
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	Description string  `xml:"description"`
	PubDate     string  `xml:"pubDate"`
	Items       []Item  `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchHomeTimeline(instanceURL, token string) ([]MastodonStatus, error) {
	req, _ := http.NewRequest("GET", instanceURL+"/api/v1/timelines/home", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var statuses []MastodonStatus
	err = json.NewDecoder(resp.Body).Decode(&statuses)
	return statuses, err
}

func serveRSS(w http.ResponseWriter, r *http.Request) {
	instance := os.Getenv("MASTODON_INSTANCE")
	token := os.Getenv("MASTODON_TOKEN")
	if instance == "" || token == "" {
		http.Error(w, "Missing env vars", http.StatusInternalServerError)
		return
	}

	statuses, err := fetchHomeTimeline(instance, token)
	if err != nil {
		http.Error(w, "Failed to fetch: "+err.Error(), 500)
		return
	}

	items := []Item{}
	for _, s := range statuses {
		items = append(items, Item{
			Title:       fmt.Sprintf("%s posted", s.Account.DisplayName),
			Link:        s.URL,
			Description: s.Content,
			PubDate:     s.CreatedAt.Format(time.RFC1123Z),
		})
	}

	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       "Mastodon Home Feed",
			Link:        instance,
			Description: "Your Mastodon home timeline",
			PubDate:     time.Now().Format(time.RFC1123Z),
			Items:       items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	xml.NewEncoder(w).Encode(rss)
}

func main() {
	http.HandleFunc("/feed.rss", serveRSS)
	log.Println("Serving RSS on :9000/feed.rss")
	log.Fatal(http.ListenAndServe(":9000", nil))
}