package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Command struct {
	Name    string `json:"name"`
	Dir     string `json:"dir"`
	Script  string `json:"script"`
	Timeout string `json:"timeout"`
}

type Config struct {
	Key      string
	Commands []*Command `json:"commands"`
}

var config *Config
var appByName map[string]*Command

var MaxTimeout = 10 * time.Minute

func main() {
	var (
		listenAddr = flag.String("listen", "localhost:8080", "address to listen on")
		configPath = flag.String("config", "config.json", "path to config.json")
	)
	flag.Parse()
	log.SetPrefix("execapi: ")

	if err := config.loadConfig(*configPath); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/run/", handleRun)

	svr := &http.Server{
		Addr:         *listenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: MaxTimeout, // hold for long deploys
	}
	log.Printf("Server running at http://%s", *listenAddr)
	log.Fatal(svr.ListenAndServe())
}

func (c *Config) loadConfig(configFile string) error {
	dat, err := os.ReadFile(configFile)
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
	if len(config.Commands) == 0 {
		return fmt.Errorf("no commands found in config.json. Please add at least one commands, see config.json.sample")
	}
	appByName = make(map[string]*Command, len(config.Commands))
	for _, app := range config.Commands {
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
	command, exists := appByName[appName]
	if !exists {
		http.Error(w, fmt.Sprintf("App '%s' not found", appName), http.StatusNotFound)
		return
	}
	verbose := r.URL.Query().Get("verbose") == "1"

	timeout, err := time.ParseDuration(command.Timeout)
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

	cmdParts := strings.Split(command.Script, " ")
	var cmd *exec.Cmd
	if len(cmdParts) == 1 {
		cmd = exec.CommandContext(ctx, cmdParts[0])
	} else {
		cmd = exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	}
	cmd.Dir = command.Dir

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
	log.Printf("Execution successful for %s, output:\n%s", command.Name, string(out))
	w.WriteHeader(http.StatusOK)
	if verbose {
		w.Write(out)
	}
}
