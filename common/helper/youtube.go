package helper

import (
	"net/url"
	"strings"
)

// IsYouTubeURL reports whether s is a YouTube video URL.
func IsYouTubeURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	host := strings.TrimPrefix(strings.TrimPrefix(u.Host, "www."), "m.")
	return host == "youtube.com" || host == "youtu.be"
}
