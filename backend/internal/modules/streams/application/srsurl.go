package application

import "strings"

// srsHTTPURL derives the SRS HTTP server base URL from the SRS HTTP API URL.
// Example: http://srs:1985 (API) → http://srs:8080 (HTTP/HLS server).
// Used by the recording and live-thumbnail capture to fetch HLS playlists.
func srsHTTPURL(srsAPIURL string) string {
	const fallback = "http://localhost:8080"
	if srsAPIURL == "" {
		return fallback
	}
	if httpURL := strings.Replace(srsAPIURL, ":1985", ":8080", 1); httpURL != srsAPIURL {
		return httpURL
	}
	return srsAPIURL + ":8080"
}
