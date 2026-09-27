/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package platforms

import (
"context"
"errors"
"fmt"
"io"
"net/http"
"os"
"strings"
"time"

"github.com/Laky-64/gologging"
"github.com/amarnathcjd/gogram/telegram"

state "main/internal/core/models"
)

const PlatformCustomApi state.PlatformName = "CustomApi"

var (
customAPIURL = "http://127.0.0.1:8001"
customAPIKey = ""
)

type CustomApiPlatform struct {
name   state.PlatformName
client *http.Client
}

func init() {
if u := strings.TrimSpace(os.Getenv("MUSIC_API_URL")); u != "" {
customAPIURL = strings.TrimRight(u, "/")
}
customAPIKey = strings.TrimSpace(os.Getenv("MUSIC_API_KEY"))

Register(85, &CustomApiPlatform{
name:   PlatformCustomApi,
client: &http.Client{Timeout: 120 * time.Second},
})
}

func (c *CustomApiPlatform) Name() state.PlatformName { return c.name }

func (c *CustomApiPlatform) CanGetTracks(_ string) bool { return false }

func (c *CustomApiPlatform) GetTracks(_ string, _ bool) ([]*state.Track, error) {
return nil, errors.New("customapi is a download-only platform")
}

func (c *CustomApiPlatform) CanDownload(source state.PlatformName) bool {
return source == PlatformYouTube
}

func (c *CustomApiPlatform) CanSearch() bool { return false }

func (c *CustomApiPlatform) Search(_ string, _ bool) ([]*state.Track, error) {
return nil, nil
}

func (c *CustomApiPlatform) Download(
ctx context.Context,
track *state.Track,
_ *telegram.NewMessage,
) (string, error) {
if f := findFile(track); f != "" {
gologging.Debug("CustomApi: cache hit -> " + f)
return f, nil
}

ext := ".opus"
videoParam := "false"
if track.Video {
ext = ".mp4"
videoParam = "true"
}

endpoint := fmt.Sprintf(
"%s/download?video_id=%s&video=%s",
customAPIURL, track.ID, videoParam,
)

req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
if err != nil {
return "", fmt.Errorf("build request: %w", err)
}
req.Header.Set("x-api-key", customAPIKey)

gologging.DebugF("CustomApi: requesting %s (video=%s)", track.ID, videoParam)

resp, err := c.client.Do(req)
if err != nil {
return "", fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
body, _ := io.ReadAll(resp.Body)
return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

path := getPath(track, ext)
f, err := os.Create(path)
if err != nil {
return "", fmt.Errorf("create file: %w", err)
}
defer f.Close()

if _, err := io.Copy(f, resp.Body); err != nil {
os.Remove(path)
return "", fmt.Errorf("write file: %w", err)
}

if !fileExists(path) {
os.Remove(path)
return "", errors.New("empty file after download")
}

gologging.InfoF("CustomApi: downloaded %s -> %s", track.ID, path)
return path, nil
}
