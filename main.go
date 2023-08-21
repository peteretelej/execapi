package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	key  string
	Apps []App `json:"apps"`
}

var config *Config
var appByName map[string]*App

var MaxTimeout = 10 * time.Minute

func main() {
	log.SetPrefix("godeploy: ")
	deployKey := os.Getenv("GODEPLOY_KEY")
	if deployKey == "" {
		log.Fatal("GODEPLOY_KEY environment variable not set")
	}
	if err := config.loadConfig(); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	config.key = deployKey

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/deploy/", handleDeploy)
	fmt.Println("Server running at http://localhost:8080")

	svr := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: MaxTimeout, // hold for long deploys
	}
	log.Fatal(svr.ListenAndServe())
}

func (c *Config) loadConfig() error {
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatal()
	}
	defer configFile.Close()

	byteValue, err := io.ReadAll(configFile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(byteValue, &config); err != nil {
		return fmt.Errorf("failed to parse config json: %v", err)
	}
	if len(config.Apps) == 0 {
		return fmt.Errorf("no apps found in config.json. Please add at least one app, see config.json.sample")
	}
	appByName = make(map[string]*App, len(config.Apps))
	for _, app := range config.Apps {
		appByName[app.Name] = &app
	}
	return nil
}

func handleDeploy(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+config.key {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	appName := r.URL.Path[len("/deploy/"):]
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
		http.Error(w, "Deployment timed out", http.StatusRequestTimeout)
		return
	}
	if err != nil {
		out := fmt.Sprintf("Deployment failed: %v\n%s", err, string(out))
		http.Error(w, out, http.StatusBadRequest)
		return
	}
	log.Printf("Deployed %s, out:\n%s", app.Name, string(out))
	w.WriteHeader(http.StatusOK)
	if verbose {
		w.Write(out)
	}
}
