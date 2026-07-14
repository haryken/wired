package mods

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/os-vector/wired/vars"
)

type Photos struct {
	vars.Modification
}

func NewPhotos() *Photos {
	return &Photos{}
}

func (modu *Photos) Name() string {
	return "Photos"
}

func (modu *Photos) Description() string {
	return "Browse and delete photos taken by Vector."
}

func (modu *Photos) Load() error {
	return nil
}

type photosInfoResp struct {
	PhotoInfos []struct {
		PhotoID      uint32 `json:"photo_id"`
		TimestampUTC uint32 `json:"timestamp_utc"`
	} `json:"photo_infos"`
}

type photoBlobResp struct {
	Success bool   `json:"success"`
	Image   string `json:"image"`
}

func (m *Photos) HTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/mods/Photos/list":
		m.list(w, r)
	case "/api/mods/Photos/thumb":
		m.serveImage(w, r, "/v1/thumbnail")
	case "/api/mods/Photos/image":
		m.serveImage(w, r, "/v1/photo")
	case "/api/mods/Photos/delete":
		m.deletePhoto(w, r)
	default:
		vars.HTTPError(w, r, "404 not found")
	}
}

func (m *Photos) list(w http.ResponseWriter, r *http.Request) {
	raw, err := cloudPostJSON("/v1/photos_info", []byte("{}"))
	if err != nil {
		vars.HTTPError(w, r, err.Error())
		return
	}
	var info photosInfoResp
	if err := json.Unmarshal(raw, &info); err != nil {
		vars.HTTPError(w, r, "bad photos_info json")
		return
	}
	type item struct {
		ID        uint32 `json:"id"`
		Timestamp uint32 `json:"timestamp_utc"`
	}
	out := make([]item, 0, len(info.PhotoInfos))
	for _, p := range info.PhotoInfos {
		out = append(out, item{ID: p.PhotoID, Timestamp: p.TimestampUTC})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp > out[j].Timestamp
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (m *Photos) serveImage(w http.ResponseWriter, r *http.Request, path string) {
	idStr := strings.TrimSpace(r.URL.Query().Get("id"))
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 {
		vars.HTTPError(w, r, "id required")
		return
	}
	body := fmt.Sprintf(`{"photo_id":%d}`, id)
	raw, err := cloudPostJSON(path, []byte(body))
	if err != nil {
		vars.HTTPError(w, r, err.Error())
		return
	}
	var resp photoBlobResp
	if err := json.Unmarshal(raw, &resp); err != nil || !resp.Success || resp.Image == "" {
		vars.HTTPError(w, r, "photo not found")
		return
	}
	img, err := base64.StdEncoding.DecodeString(resp.Image)
	if err != nil {
		img, err = base64.RawStdEncoding.DecodeString(resp.Image)
		if err != nil {
			vars.HTTPError(w, r, "bad image encoding")
			return
		}
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(img)
}

func (m *Photos) deletePhoto(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(r.URL.Query().Get("id"))
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 {
		vars.HTTPError(w, r, "id required")
		return
	}
	body := fmt.Sprintf(`{"photo_id":%d}`, id)
	if _, err := cloudPostJSON("/v1/delete_photo", []byte(body)); err != nil {
		vars.HTTPError(w, r, err.Error())
		return
	}
	vars.HTTPSuccess(w, r)
}
