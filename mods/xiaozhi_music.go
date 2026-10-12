package mods

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var musicTestClient = &http.Client{Timeout: 12 * time.Second}

// musicTestResult is what the Bot Settings "Thử server" button shows.
type musicTestResult struct {
	Status   string `json:"status"`
	URL      string `json:"url"`
	Query    string `json:"query"`
	SearchOK bool   `json:"search_ok"`
	Title    string `json:"title,omitempty"`
	StreamOK bool   `json:"stream_ok"`
	Error    string `json:"error,omitempty"`
}

// testMusicServer runs the same two calls vic-cloud makes for self.music.play:
// search, then the first bytes of the MP3 stream for the top hit.
func testMusicServer(base, query string) musicTestResult {
	res := musicTestResult{Status: "success", URL: base, Query: query}

	var sr struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Data    []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"data"`
	}
	code, body, err := musicTestGet(base + "/api/search?q=" + url.QueryEscape(query) + "&limit=1")
	if err != nil {
		res.Error = "search: " + err.Error()
		return res
	}
	if jerr := json.Unmarshal(body, &sr); jerr != nil || code != http.StatusOK || !sr.Success {
		msg := sr.Error
		if msg == "" {
			msg = shortMusicMsg(string(body))
		}
		res.Error = fmt.Sprintf("search HTTP %d: %s", code, msg)
		return res
	}
	if len(sr.Data) == 0 || strings.TrimSpace(sr.Data[0].ID) == "" {
		res.Error = "search: no results"
		return res
	}
	res.SearchOK = true
	res.Title = sr.Data[0].Title

	req, err := http.NewRequest(http.MethodGet,
		base+"/api/stream/mp3?id="+url.QueryEscape(sr.Data[0].ID)+"&format=mp3", nil)
	if err != nil {
		res.Error = "stream: " + err.Error()
		return res
	}
	resp, err := musicTestClient.Do(req)
	if err != nil {
		res.Error = "stream: " + err.Error()
		return res
	}
	defer resp.Body.Close()
	head, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		msg := string(head)
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(head, &e) == nil && e.Error != "" {
			msg = e.Error
		}
		res.Error = fmt.Sprintf("stream HTTP %d: %s", resp.StatusCode, shortMusicMsg(msg))
		return res
	}
	if len(head) == 0 {
		res.Error = "stream: empty body"
		return res
	}
	res.StreamOK = true
	return res
}

func musicTestGet(u string) (int, []byte, error) {
	resp, err := musicTestClient.Get(u)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	return resp.StatusCode, body, err
}

func shortMusicMsg(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 160 {
		return string(r[:160]) + "…"
	}
	return s
}
