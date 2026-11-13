package main

import (
	"context"
	"log"
	"os"
	"time"

	yily "github.com/yannick2009/yily-sdk-go"
)

type Config struct {
	Host      string        `yily:"DB_HOST"`
	Port      int           `yily:"DB_PORT"`
	Debug     bool          `yily:"DEBUG"`
	Timeout   time.Duration `yily:"TIMEOUT"`
	APIToken  string        `yily:"API_TOKEN"`
	LegacyKey string        `yily:"LEGACY_KEY,version=2"` // LegacyKey loads version 2 of LEGACY_KEY instead of the active version.
}

func main() {
	ctx := context.Background()

	// Create the client once and reuse it for every call.
	client, err := yily.NewClient(yily.Config{
		BaseURL:       "http://localhost:16529",
		EnvironmentID: "00000000-0000-0000-0000-000000000000",
		Token:         os.Getenv("YILY_TOKEN"),
	})
	if err != nil {
		log.Fatalf("client error: %v", err)
	}

	// The examples below are independent. In a real application, use the one
	// that fits your need; they do not depend on each other.

	// Load: fill a typed struct from the fields tagged with `yily`.
	var cfg Config
	if err := client.Load(ctx, &cfg); err != nil {
		log.Fatalf("load error: %v", err)
	}
	log.Printf("configuration loaded: host=%s port=%d debug=%t timeout=%s legacy_key_loaded=%t",
		cfg.Host, cfg.Port, cfg.Debug, cfg.Timeout, cfg.LegacyKey != "")

	// Get: read a single variable by name. This is useful when the name is
	// only known at runtime. Do not log secret values in production.
	apiToken, err := client.Get(ctx, "API_TOKEN")
	if err != nil {
		log.Fatalf("get error: %v", err)
	}
	log.Printf("API_TOKEN retrieved (%d characters)", len(apiToken))

	// GetVersion: read a specific version of a single variable.
	legacyToken, err := client.GetVersion(ctx, "API_TOKEN", 1)
	if err != nil {
		log.Fatalf("get versioned error: %v", err)
	}
	log.Printf("API_TOKEN version 1 retrieved (%d characters)", len(legacyToken))

	// GetAll: retrieve every variable of the environment as a map, when the
	// variable names are not known in advance.
	all, err := client.GetAll(ctx)
	if err != nil {
		log.Fatalf("getAll error: %v", err)
	}
	log.Printf("retrieved %d variables", len(all))
}
