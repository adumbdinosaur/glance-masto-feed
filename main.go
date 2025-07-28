package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
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
	Avatar      string `json:"avatar"`
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
	Reblog           *MastodonStatus `json:"reblog"`
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

	http.HandleFunc("/feed.html", func(w http.ResponseWriter, r *http.Request) {
		statuses, err := fetchTimeline(server, token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<title>Mastodon Feed</title>
			<style>
				body { font-family: sans-serif; font-size: 14px; background: #111; color: #eee; padding: 10px; }
				a { color: #8ecfff; text-decoration: none; }
				img { max-width: 100%; height: auto; border-radius: 6px; margin-top: 5px; }
				.status { margin-bottom: 2em; border-bottom: 1px solid #333; padding-bottom: 1em; }
				.status strong { color: #ffda7b; }
				.status small { color: #aaa; }
				.avatar { width: 48px; height: 48px; border-radius: 50%; vertical-align: middle; margin-right: 8px; }
			</style>
		</head>
		<body>
			{{range .}}
			<div class="status">
				<img src="{{.Avatar}}" class="avatar" alt="avatar">
				<strong>{{.Title}}</strong><br>
				<small>{{.PubDate}}</small>
				<div>{{.Description}}</div>
			</div>
			{{end}}
		</body>
		</html>`
		t := template.Must(template.New("feed").Parse(tmpl))
		t.Execute(w, flattenToHtmlItems(statuses))
	})

	port := ":9000"
	fmt.Println("Serving feeds on", port)
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
	items := flattenToItems(statuses)
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

type HtmlItem struct {
	Title       string
	Link        string
	Description template.HTML
	PubDate     string
	Avatar      string
}

func flattenToHtmlItems(statuses []MastodonStatus) []HtmlItem {
	items := []HtmlItem{}
	for _, s := range statuses {
		post := &s
		displayName := s.Account.DisplayName
		avatar := s.Account.Avatar
		if s.Reblog != nil {
			displayName = fmt.Sprintf("%s ↻ %s", s.Account.DisplayName, s.Reblog.Account.DisplayName)
			post = s.Reblog
			avatar = s.Reblog.Account.Avatar
		}
		item := HtmlItem{
			Title:       fmt.Sprintf("%s posted", displayName),
			Link:        post.URL,
			Description: buildHTMLDescription(*post),
			PubDate:     post.CreatedAt.Format(time.RFC1123Z),
			Avatar:      avatar,
		}
		items = append(items, item)
	}
	return items
}

func flattenToItems(statuses []MastodonStatus) []Item {
	items := []Item{}
	for _, s := range statuses {
		post := &s
		displayName := s.Account.DisplayName
		if s.Reblog != nil {
			displayName = fmt.Sprintf("%s ↻ %s", s.Account.DisplayName, s.Reblog.Account.DisplayName)
			post = s.Reblog
		}
		item := Item{
			Title:       fmt.Sprintf("%s posted", displayName),
			Link:        post.URL,
			Description: buildRSSDescription(*post),
			PubDate:     post.CreatedAt.Format(time.RFC1123Z),
		}
		items = append(items, item)
	}
	return items
}

func buildHTMLDescription(s MastodonStatus) template.HTML {
	var sb strings.Builder

	if s.Content != "" {
		sb.WriteString(s.Content)
	}

	for _, media := range s.MediaAttachments {
		if media.Type == "image" {
			sb.WriteString(fmt.Sprintf(`<br><img src="%s" alt="%s">`, media.URL, media.Description))
		}
	}

	if s.Content == "" && len(s.MediaAttachments) == 0 {
		sb.WriteString("<em>No content</em>")
	}

	return template.HTML(sb.String())
}

func buildRSSDescription(s MastodonStatus) string {
	var sb strings.Builder
	sb.WriteString("<![CDATA[")

	if s.Content != "" {
		sb.WriteString(s.Content)
	}

	for _, media := range s.MediaAttachments {
		if media.Type == "image" {
			sb.WriteString(fmt.Sprintf(`<br><img src="%s" alt="%s">`, media.URL, media.Description))
		}
	}

	if s.Content == "" && len(s.MediaAttachments) == 0 {
		sb.WriteString("<em>No content</em>")
	}

	sb.WriteString("]]>")
	return sb.String()
}
