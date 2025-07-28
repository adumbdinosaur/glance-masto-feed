package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type MastodonAccount struct {
	DisplayName string `json:"display_name"`
	Acct        string `json:"acct"`
}

type MastodonMedia struct {
	Type        string `json:"type"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type MastodonStatus struct {
	Content          string          `json:"content"`
	CreatedAt        time.Time       `json:"created_at"`
	Account          MastodonAccount `json:"account"`
	URL              string          `json:"url"`
	MediaAttachments []MastodonMedia `json:"media_attachments"`
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func main() {
	server := os.Getenv("MASTODON_INSTANCE")
	token := os.Getenv("MASTODON_TOKEN")
	if server == "" || token == "" {
		log.Fatal("MASTODON_INSTANCE and MASTODON_TOKEN must be set")
	}

	http.HandleFunc("/feed.rss", func(w http.ResponseWriter, r *http.Request) {
		statuses, err := fetchTimeline(server, token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		feed := generateRSS(statuses)
		w.Header().Set("Content-Type", "application/rss+xml")
		xml.NewEncoder(w).Encode(feed)
	})

	port := ":9000"
	fmt.Println("Serving RSS feed on", port)
	http.ListenAndServe(port, nil)
}

func fetchTimeline(server, token string) ([]MastodonStatus, error) {
	req, err := http.NewRequest("GET", server+"/api/v1/timelines/home", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var statuses []MastodonStatus
	err = json.Unmarshal(body, &statuses)
	if err != nil {
		return nil, err
	}

	return statuses, nil
}

func generateRSS(statuses []MastodonStatus) RSS {
	items := []Item{}
	for _, s := range statuses {
		item := Item{
			Title:       fmt.Sprintf("%s posted", s.Account.DisplayName),
			Link:        s.URL,
			Description: buildDescription(s),
			PubDate:     s.CreatedAt.Format(time.RFC1123Z),
		}
		items = append(items, item)
	}

	return RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       "Mastodon Home Feed",
			Link:        "https://meow.social",
			Description: "Your Mastodon home timeline",
			PubDate:     time.Now().Format(time.RFC1123Z),
			Items:       items,
		},
	}
}

func buildDescription(s MastodonStatus) string {
	var sb strings.Builder
	sb.WriteString("<![CDATA[")

	if s.Content != "" {
		sb.WriteString(s.Content)
	}

	for _, media := range s.MediaAttachments {
		if media.Type == "image" {
			sb.WriteString(fmt.Sprintf(`<br><img src="%s" alt="%s" style="max-width:100%%;">`, media.URL, media.Description))
		}
	}

	if s.Content == "" && len(s.MediaAttachments) == 0 {
		sb.WriteString("<em>No content</em>")
	}

	sb.WriteString("]]>")
	return sb.String()
}
