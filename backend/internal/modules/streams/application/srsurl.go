package application

import "net/url"

// srsHTTPURL derives the SRS HTTP server base URL from the SRS HTTP API URL.
// Example: http://srs:1985 (API) → http://srs:8080 (HTTP/HLS server).
// Used by the recording and live-thumbnail capture to fetch HLS playlists.
func srsHTTPURL(srsAPIURL string) string {
	const fallback = "http://localhost:8080"
	if srsAPIURL == "" {
		return fallback
	}
	u, err := url.Parse(srsAPIURL)
	if err != nil || u.Host == "" {
		return fallback
	}
	u.Host = u.Hostname() + ":8080"
	return u.String()
}
