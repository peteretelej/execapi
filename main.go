package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type App struct {
	Name    string `json:"name"`
	Dir     string `json:"dir"`
	Script  string `json:"script"`
	Timeout string `json:"timeout"`
}

type Config struct {
	Key  string
	Apps []*App `json:"apps"`
}

var config *Config
var appByName map[string]*App

var MaxTimeout = 10 * time.Minute

func main() {
	log.SetPrefix("execapi: ")
	if err := config.loadConfig(); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/run/", handleRun)
	fmt.Println("Server running at http://localhost:8080")

	log.Printf("apps: %+v", appByName)
	svr := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: MaxTimeout, // hold for long deploys
	}
	log.Fatal(svr.ListenAndServe())
}

func (c *Config) loadConfig() error {
	dat, err := os.ReadFile("config.json")
	if err != nil {
		return err
	}

	if err := json.Unmarshal(dat, &config); err != nil {
		return fmt.Errorf("failed to parse config json: %v", err)
	}
	if config.Key == "" {
		return errors.New("no key found in config.json. Please add a key, see config.json.sample")
	}
	if config.Key == "EXECAPI_KEY_HERE" {
		return errors.New("please add a custom key to the config.json.")
	}
	if len(config.Apps) == 0 {
		return fmt.Errorf("no apps found in config.json. Please add at least one app, see config.json.sample")
	}
	appByName = make(map[string]*App, len(config.Apps))
	for _, app := range config.Apps {
		appByName[app.Name] = app
	}
	return nil
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+config.Key {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	appName := r.URL.Path[len("/run/"):]
	app, exists := appByName[appName]
	if !exists {
		http.Error(w, fmt.Sprintf("App '%s' not found", appName), http.StatusNotFound)
		return
	}
	verbose := r.URL.Query().Get("verbose") == "1"

	timeout, err := time.ParseDuration(app.Timeout)
	if err != nil {
		http.Error(w, "Invalid timeout", http.StatusBadRequest)
		return
	}
	if timeout > MaxTimeout {
		http.Error(w, fmt.Sprintf("Timeout exceeds max allowed timeout of %s", MaxTimeout), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmdParts := strings.Split(app.Script, " ")
	var cmd *exec.Cmd
	if len(cmdParts) == 1 {
		cmd = exec.CommandContext(ctx, cmdParts[0])
	} else {
		cmd = exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	}
	cmd.Dir = app.Dir

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		http.Error(w, "Execution timed out", http.StatusRequestTimeout)
		return
	}
	if err != nil {
		out := fmt.Sprintf("Execution failed: %v\n%s", err, string(out))
		http.Error(w, out, http.StatusBadRequest)
		return
	}
	log.Printf("Execution successful for %s, output:\n%s", app.Name, string(out))
	w.WriteHeader(http.StatusOK)
	if verbose {
		w.Write(out)
	}
}
