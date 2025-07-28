# Glance Mastodon Feed

A simple Go application that fetches your Mastodon home timeline and serves it as RSS or HTML feeds.

## Features

- Fetches posts from your Mastodon home timeline
- Serves posts as RSS feed (`/feed.rss`) for feed readers
- Serves posts as HTML feed (`/feed.html`) with interactive buttons
- **NEW**: Click-to-interact buttons that open posts in your home instance for easy replying, liking, and boosting!

## Environment Variables

- `MASTODON_INSTANCE`: Your Mastodon server (e.g. `https://fosstodon.org`)
- `MASTODON_TOKEN`: Your Mastodon access token
- `HOME_INSTANCE`: Your preferred Mastodon instance for interactions (e.g. `mastodon.social`, `fosstodon.org`)

## Usage

1. Set your environment variables:
   ```bash
   export MASTODON_INSTANCE="https://your-instance.social"
   export MASTODON_TOKEN="your-access-token"
   export HOME_INSTANCE="your-home-instance.social"  # Required for interactions
   ```

2. Run the application:
   ```bash
   go run main.go
   ```

3. Visit the feeds:
   - HTML feed with interaction buttons: http://localhost:9000/feed.html
   - RSS feed: http://localhost:9000/feed.rss

## Interactive Features

The HTML feed now includes:
- **"Find on home instance"** link in the post header - Opens the post URL in your home instance's search
- **Reply** button (💬) - Opens your home instance's compose window with `@username` pre-filled for replying
- **Like** button (❤️) - Uses the API to directly like the post (requires write token)
- **Boost** button (🚀) - Uses the API to directly boost/reblog the post (requires write token)

### API vs Home Instance Actions

- **Like & Boost**: Use your API token to perform the action directly on your timeline instance
- **Reply**: Opens your home instance's compose window since replies require more user input

### Token Permissions

For the Like and Boost buttons to work, your `MASTODON_TOKEN` needs **write permissions**. If you only have a read-only token:
- Like and Boost buttons will show an error
- Reply button will still work (opens in home instance)
- All posts will still be visible

## How it Works

- **Reading Timeline**: Uses the Mastodon API to fetch your home timeline
- **Direct Actions**: Like and Boost buttons use the API directly with your token
- **Replies**: Opens your home instance's compose window for complex interactions
- **Federation**: Handles cross-instance interactions seamlessly

## Using with Glance Dashboard

This application is designed to be embedded in [Glance](https://github.com/glanceapp/glance) dashboards. Each user must set their own `HOME_INSTANCE` environment variable to their preferred Mastodon instance for interactions.

Example Glance configuration:
```yaml
widgets:
  - type: iframe
    source: "http://localhost:9000/feed.html"
    title: "Mastodon Feed"
    height: 600
```

**Important**: Make sure each deployment sets their own `HOME_INSTANCE` environment variable. There is no default to prevent overwhelming any single instance with traffic.
