package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
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
	ID               string          `json:"id"`
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
	homeInstance := os.Getenv("HOME_INSTANCE")
	if server == "" || token == "" || homeInstance == "" {
		log.Fatal("Required environment variables:\n" +
			"  MASTODON_INSTANCE - Your Mastodon server (e.g. https://fosstodon.org)\n" +
			"  MASTODON_TOKEN - Your Mastodon access token\n" +
			"  HOME_INSTANCE - Your home instance for interactions (e.g. mastodon.social)")
	}

	fmt.Printf("Using home instance: %s\n", homeInstance)

	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		testURL := "https://fosstodon.org/@alice/123456789"
		testAcct := "alice"
		encodedURL := url.QueryEscape(testURL)

		shareURL := fmt.Sprintf("https://%s/share?text=%s", homeInstance, encodedURL)
		replyShareURL := fmt.Sprintf("https://%s/share?text=@%s%%20%s", homeInstance, testAcct, encodedURL)
		searchURL := fmt.Sprintf("https://%s/search?q=%s", homeInstance, encodedURL)

		fmt.Fprintf(w, `
		<h1>Debug Information</h1>
		<p><strong>Home Instance:</strong> %s</p>
		<p><strong>Test Post URL:</strong> %s</p>
		<p><strong>Encoded URL:</strong> %s</p>
		<br>
		<p><strong>Current Methods:</strong></p>
		<p><a href="%s" target="_blank">📤 Share Method (General)</a></p>
		<p><a href="%s" target="_blank">💬 Reply Share Method</a></p>
		<p><a href="%s" target="_blank">🔍 Search Method</a></p>
		<br>
		<p><strong>Instructions:</strong> Click the links above to test them. The Share method should open your home instance's compose window with the post URL pre-filled.</p>
		`, homeInstance, testURL, encodedURL, shareURL, replyShareURL, searchURL)
	})

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
				a:hover { text-decoration: underline; }
				img { max-width: 100%; height: auto; border-radius: 6px; margin-top: 5px; }
				.status { margin-bottom: 2em; border-bottom: 1px solid #333; padding-bottom: 1em; }
				.status strong { color: #ffda7b; }
				.status small { color: #aaa; }
				.avatar { width: 48px; height: 48px; border-radius: 50%; vertical-align: middle; margin-right: 8px; }
				.actions { margin-top: 10px; }
				.action-btn { 
					display: inline-block; 
					background: #1d4ed8; 
					color: white; 
					padding: 4px 8px; 
					border-radius: 4px; 
					margin-right: 8px; 
					font-size: 12px;
					text-decoration: none;
				}
				.action-btn:hover { background: #2563eb; color: white; text-decoration: none; }
				.reply-btn { background: #16a34a; }
				.reply-btn:hover { background: #15803d; }
				.boost-btn { background: #f59e0b; }
				.boost-btn:hover { background: #d97706; }
				.like-btn { background: #dc2626; }
				.like-btn:hover { background: #b91c1c; }
				.success { color: #16a34a; font-weight: bold; }
				.error { color: #dc2626; font-weight: bold; }
				.action-btn { 
					display: inline-block; 
					background: #1d4ed8; 
					color: white; 
					padding: 4px 8px; 
					border-radius: 4px; 
					margin-right: 8px; 
					font-size: 12px;
					text-decoration: none;
					cursor: pointer;
					border: none;
				}
			</style>
			<script>
				async function performAction(url, actionName, button) {
					try {
						button.disabled = true;
						button.textContent = actionName + 'ing...';
						
						const response = await fetch(url, { method: 'POST' });
						const result = await response.json();
						
						if (response.ok && result.success) {
							button.textContent = '✓ ' + actionName + 'd';
							button.className = button.className + ' success';
						} else {
							throw new Error(result.error || 'Failed to ' + actionName.toLowerCase());
						}
					} catch (error) {
						button.textContent = '✗ ' + error.message;
						button.className = button.className + ' error';
						button.disabled = false;
						console.error('Action failed:', error);
						
						setTimeout(() => {
							button.textContent = button.textContent.includes('Like') ? '❤️ Like' : '🚀 Boost';
							button.className = button.className.replace(' error', '');
							button.disabled = false;
						}, 3000);
					}
				}
			</script>
		</head>
		<body>
			<h1>Mastodon Feed</h1>
			<p><small>Like/Boost buttons use API directly. Reply finds the post on your home instance: <strong>{{.HomeInstance}}</strong> | <a href="/debug">Debug Info</a></small></p>
			{{range .Statuses}}
			<div class="status">
				<img src="{{.Avatar}}" class="avatar" alt="avatar">
				<strong>{{.Title}}</strong><br>
				<small>{{.PubDate}} | <a href="{{.HomeLink}}" target="_blank">� Share/Interact</a></small>
				<div>{{.Description}}</div>
				<div class="actions">
					<a href="{{.ReplyLink}}" target="_blank" class="action-btn reply-btn">💬 Find & Reply</a>
					<button onclick="performAction('{{.LikeLink}}', 'Like', this)" class="action-btn like-btn">❤️ Like</button>
					<button onclick="performAction('{{.BoostLink}}', 'Boost', this)" class="action-btn boost-btn">🚀 Boost</button>
				</div>
			</div>
			{{end}}
		</body>
		</html>`
		t := template.Must(template.New("feed").Parse(tmpl))

		data := struct {
			HomeInstance string
			Statuses     []HtmlItem
		}{
			HomeInstance: homeInstance,
			Statuses:     flattenToHtmlItems(statuses, homeInstance),
		}

		t.Execute(w, data)
	})

	http.HandleFunc("/api/like/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		postID := strings.TrimPrefix(r.URL.Path, "/api/like/")
		if postID == "" {
			http.Error(w, "Post ID required", http.StatusBadRequest)
			return
		}

		fmt.Printf("Attempting to like post %s on %s\n", postID, server)
		err := likePost(server, token, postID)
		if err != nil {
			fmt.Printf("Error liking post: %v\n", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf(`{"success": false, "error": "%s"}`, err.Error())))
			return
		}

		fmt.Printf("Successfully liked post %s\n", postID)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success": true, "action": "liked"}`))
	})

	http.HandleFunc("/api/boost/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		postID := strings.TrimPrefix(r.URL.Path, "/api/boost/")
		if postID == "" {
			http.Error(w, "Post ID required", http.StatusBadRequest)
			return
		}

		fmt.Printf("Attempting to boost post %s on %s\n", postID, server)
		err := boostPost(server, token, postID)
		if err != nil {
			fmt.Printf("Error boosting post: %v\n", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf(`{"success": false, "error": "%s"}`, err.Error())))
			return
		}

		fmt.Printf("Successfully boosted post %s\n", postID)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success": true, "action": "boosted"}`))
	})

	http.HandleFunc("/api/reply/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		postID := strings.TrimPrefix(r.URL.Path, "/api/reply/")
		if postID == "" {
			http.Error(w, "Post ID required", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var replyData struct {
			Text string `json:"text"`
		}
		err = json.Unmarshal(body, &replyData)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		fmt.Printf("💬 Replying to post %s with: %s\n", postID, replyData.Text)
		err = replyToPost(server, token, postID, replyData.Text)
		if err != nil {
			fmt.Printf("❌ Reply failed: %v\n", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf(`{"success": false, "error": "%s"}`, err.Error())))
			return
		}

		fmt.Printf("✅ Replied to post %s\n", postID)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success": true, "action": "replied"}`))
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
			Link:        "https://mastodon.social",
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
	PostID      string
	HomeLink    string
	ReplyLink   string
	LikeLink    string
	BoostLink   string
}

func flattenToHtmlItems(statuses []MastodonStatus, homeInstance string) []HtmlItem {
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

		encodedURL := url.QueryEscape(post.URL)

		replyLink := fmt.Sprintf("https://%s/search?q=%s", homeInstance, encodedURL)

		likeLink := fmt.Sprintf("/api/like/%s", post.ID)
		boostLink := fmt.Sprintf("/api/boost/%s", post.ID)

		homeLink := fmt.Sprintf("https://%s/search?q=%s", homeInstance, encodedURL)

		item := HtmlItem{
			Title:       fmt.Sprintf("%s posted", displayName),
			Link:        post.URL,
			Description: buildHTMLDescription(*post),
			PubDate:     post.CreatedAt.Format(time.RFC1123Z),
			Avatar:      avatar,
			PostID:      post.ID,
			HomeLink:    homeLink,
			ReplyLink:   replyLink,
			LikeLink:    likeLink,
			BoostLink:   boostLink,
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

func likePost(server, token, postID string) error {
	url := fmt.Sprintf("%s/api/v1/statuses/%s/favourite", server, postID)

	fmt.Printf("❤️ Liking post %s...\n", postID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Like failed: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Like failed (status %d): %s\n", resp.StatusCode, string(body))
		return fmt.Errorf("failed to like post: %s (status %d)", body, resp.StatusCode)
	}

	fmt.Printf("✅ Liked post %s\n", postID)
	return nil
}

func boostPost(server, token, postID string) error {
	url := fmt.Sprintf("%s/api/v1/statuses/%s/reblog", server, postID)

	fmt.Printf("� Boosting post %s...\n", postID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Boost failed: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Boost failed (status %d): %s\n", resp.StatusCode, string(body))
		return fmt.Errorf("failed to boost post: %s (status %d)", body, resp.StatusCode)
	}

	fmt.Printf("✅ Boosted post %s\n", postID)
	return nil
}

func replyToPost(server, token, postID, text string) error {
	url := fmt.Sprintf("%s/api/v1/statuses", server)

	replyData := map[string]interface{}{
		"status":         text,
		"in_reply_to_id": postID,
		"visibility":     "public",
	}

	jsonData, err := json.Marshal(replyData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to reply to post: %s (status %d)", body, resp.StatusCode)
	}

	return nil
}
