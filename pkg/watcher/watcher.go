package watcher

import (
    "io/ioutil"
    "log"
    "path/filepath"
    "strings"

    "github.com/fsnotify/fsnotify"
)

// Watch monitors a directory (mounted secret) for changes to client_id and client_secret files.
// It calls onChange whenever either file is created or modified.
func Watch(dir string, onChange func(newID, newSecret string)) {
    // Initial load
    loadAndNotify(dir, onChange)

    // Create new watcher
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Fatalf("failed to create file watcher: %v", err)
    }
    defer watcher.Close()

    // Add directory to watcher
    if err := watcher.Add(dir); err != nil {
        log.Fatalf("failed to watch directory %s: %v", dir, err)
    }

    // Process events
    for {
        select {
        case event, ok := <-watcher.Events:
            if !ok {
                return
            }
            // React only on create or write events for our secret files
            if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
                base := filepath.Base(event.Name)
                if base == "client_id" || base == "client_secret" {
                    loadAndNotify(dir, onChange)
                }
            }
        case err, ok := <-watcher.Errors:
            if !ok {
                return
            }
            log.Printf("watcher error: %v", err)
        }
    }
}

// loadAndNotify reads client_id and client_secret from the directory and invokes onChange.
func loadAndNotify(dir string, onChange func(newID, newSecret string)) {
    // Read client_id
    idBytes, err := ioutil.ReadFile(filepath.Join(dir, "client_id"))
    if err != nil {
        log.Printf("failed to read client_id: %v", err)
        return
    }
    // Read client_secret
    secretBytes, err := ioutil.ReadFile(filepath.Join(dir, "client_secret"))
    if err != nil {
        log.Printf("failed to read client_secret: %v", err)
        return
    }

    newID := strings.TrimSpace(string(idBytes))
    newSecret := strings.TrimSpace(string(secretBytes))

    onChange(newID, newSecret)
}