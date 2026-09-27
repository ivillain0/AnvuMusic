/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package main

/*
#cgo CFLAGS: -I../../
#cgo linux LDFLAGS: -L ../../ -lntgcalls -lm -lz
#cgo darwin LDFLAGS: -L ../../ -lntgcalls -lc++ -lz -lbz2 -liconv -framework AVFoundation -framework AudioToolbox -framework CoreAudio -framework QuartzCore -framework CoreMedia -framework VideoToolbox -framework AppKit -framework Metal -framework MetalKit -framework OpenGL -framework IOSurface -framework ScreenCaptureKit

// Currently is supported only dynamically linked library on Windows due to
// https://github.com/golang/go/issues/63903
#cgo windows LDFLAGS: -L../../ -lntgcalls
#include "ntgcalls/ntgcalls.h"
#include "glibc_compatibility.h"
*/
import "C"

import (
"fmt"
"net/http"
_ "net/http/pprof"
"net/url"
"os"
"os/exec"
"time"

"github.com/Laky-64/gologging"

"main/internal/config"
"main/internal/core"
"main/internal/database"
"main/internal/locales"
"main/internal/modules"
"main/internal/platforms"
)

func main() {
initLogger()
defer config.CloseLogging()

shutdownMusicAPI := startMusicAPIServer()
defer shutdownMusicAPI()

shutdownPlatforms, err := platforms.Init()
if err != nil {
gologging.Fatal("Failed to initialize platforms: " + err.Error())
}
defer shutdownPlatforms()

checkFFmpegAndFFprobe()

if err := refreshDirs(); err != nil {
gologging.Fatal("Failed to refresh directories: " + err.Error())
}

gologging.Debug("Initializing MongoDB...")

closeDB, err := database.Init(config.MongoURI)
if err != nil {
gologging.Fatal("Failed to initialize database: " + err.Error())
}
defer closeDB()

gologging.Info("Database connected successfully")

if err := locales.Load(); err != nil {
gologging.Fatal("Failed to load locales: " + err.Error())
}

gologging.Debug("Initializing clients...")

shutdownCore, err := core.Init()
if err != nil {
gologging.Fatal("Failed to initialize core: " + err.Error())
}
defer shutdownCore()

core.GetAssistantIndexFunc = database.AssistantIndex
core.F = modules.F

if err := database.RebalanceAssistantIndexes(core.Assistants.Count()); err != nil {
gologging.Fatal("Failed to rebalance Assistants: " + err.Error())
}

modules.Init(core.Bot, core.Assistants)

startHTTPServer()

core.Bot.Idle()
}

// startMusicAPIServer launches the local FastAPI download service (api-server/)
// as a child process when MUSIC_API_URL points at localhost, and waits for it
// to report healthy before continuing. It returns a shutdown func to stop it.
func startMusicAPIServer() func() {
apiURL := os.Getenv("MUSIC_API_URL")
if apiURL == "" {
apiURL = "http://127.0.0.1:8001"
}

parsed, err := url.Parse(apiURL)
if err != nil {
gologging.Warn("Invalid MUSIC_API_URL, skipping auto-start: " + err.Error())
return func() {}
}

host := parsed.Hostname()
if host != "127.0.0.1" && host != "localhost" {
gologging.Info("MUSIC_API_URL points to a remote host; not auto-starting a local server.")
return func() {}
}

port := parsed.Port()
if port == "" {
port = "8001"
}

dir := "api-server"
if _, err := os.Stat(dir); err != nil {
gologging.Warn("api-server folder not found, skipping auto-start: " + err.Error())
return func() {}
}

cmd := exec.Command("python3", "-m", "uvicorn", "main:app",
"--host", "127.0.0.1", "--port", port)
cmd.Dir = dir
cmd.Env = os.Environ()
cmd.Stdout = config.LogWriter
cmd.Stderr = config.LogWriter

if err := cmd.Start(); err != nil {
gologging.Error("Failed to start Music API server: " + err.Error())
return func() {}
}

gologging.Info(fmt.Sprintf("Music API server starting (pid %d)...", cmd.Process.Pid))

healthURL := "http://127.0.0.1:" + port + "/health"
deadline := time.Now().Add(15 * time.Second)
ready := false
for time.Now().Before(deadline) {
resp, err := http.Get(healthURL)
if err == nil {
resp.Body.Close()
if resp.StatusCode == http.StatusOK {
ready = true
break
}
}
time.Sleep(500 * time.Millisecond)
}

if ready {
gologging.Info("Music API server is ready.")
} else {
gologging.Warn("Music API server did not report healthy in time; continuing anyway (yt-dlp direct fallback still applies).")
}

return func() {
gologging.Info("Stopping Music API server...")
if cmd.Process != nil {
_ = cmd.Process.Kill()
_, _ = cmd.Process.Wait()
}
}
}

func startHTTPServer() {
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusOK)
w.Write([]byte("ok"))
})
go func() {
addr := "0.0.0.0:" + config.Port
if err := http.ListenAndServe(addr, nil); err != nil {
gologging.Error("HTTP server error: " + err.Error())
}
}()
go selfPing()
}

func selfPing() {
time.Sleep(10 * time.Second) // wait for server to start
for {
resp, err := http.Get("http://localhost:" + config.Port + "/")
if err == nil {
resp.Body.Close()
}
time.Sleep(5 * time.Minute)
}
}

func initLogger() {
gologging.SetLevel(gologging.DebugLevel)
gologging.SetOutput(config.LogWriter)

l := gologging.GetLogger("ntgcalls")
l.SetLevel(gologging.ErrorLevel)
l.SetOutput(config.LogWriter)

l = gologging.GetLogger("webrtc")
l.SetLevel(gologging.ErrorLevel)
l.SetOutput(config.LogWriter)

gologging.GetLogger("Database").SetOutput(config.LogWriter)
}

func refreshDirs() error {
dirs := []string{
"./cache",
"./downloads",
}

for _, dir := range dirs {

if err := os.RemoveAll(dir); err != nil {
return err
}

if err := os.MkdirAll(dir, 0o755); err != nil {
return err
}
}

return nil
}
