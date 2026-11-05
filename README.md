# Yily Go SDK

Official lightweight Go SDK for retrieving configuration variables from [Yily](https://github.com/yannick2009/yily).

## Requirements

- **Go**: 1.25 or later.
- **Yily Server**: Requires a Yily instance that supports the batch variable values endpoint (`GET /api/v1/environments/:environmentID/variables/values`).

## Installation

```bash
go get github.com/yannick2009/yily-sdk-go
```

## Quickstart

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/yannick2009/yily-sdk-go"
)

type AppConfig struct {
	DBHost  string        `yily:"DB_HOST"`
	DBPort  int           `yily:"DB_PORT"`
	Debug   bool          `yily:"DEBUG"`
	Timeout time.Duration `yily:"TIMEOUT"`
}

func main() {
	ctx := context.Background()

	client, err := yily.NewClient(yily.Config{
		BaseURL:       "http://localhost:16529",
		EnvironmentID: "your-environment-uuid",
		Token:         os.Getenv("YILY_TOKEN"),
	})
	if err != nil {
		log.Fatalf("failed to initialize Yily client: %v", err)
	}

	// 1. Load configuration into a struct
	var cfg AppConfig
	if err := client.Load(ctx, &cfg); err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	log.Println("Yily configuration loaded")

	// 2. Retrieve all active variables as a map without knowing names in advance
	allVars, err := client.GetAll(ctx)
	if err != nil {
		log.Fatalf("failed to get all variables: %v", err)
	}
	_ = allVars // map[string]string

	// 3. Retrieve an individual variable by name
	apiKey, err := client.Get(ctx, "API_KEY")
	if err != nil {
		log.Fatalf("failed to get API_KEY: %v", err)
	}
	if apiKey == "" {
		log.Fatal("API_KEY is empty")
	}

	// 4. Retrieve a specific version of a variable
	legacyKey, err := client.Get(ctx, "API_KEY", yily.Version(1))
	if err != nil {
		log.Fatalf("failed to get legacy key: %v", err)
	}
	_ = legacyKey // Use the value in the application; do not log secrets.
}
```

Use a read-scoped PAT for applications that only need to load configuration. PATs are created and managed through Yily's web interface. `Load` makes one request for active values; if fields request historical versions, it makes one request for each distinct version number. `GetAll` makes one request to list the environment's variables and a second batch request to fetch their active values.

Use HTTPS for deployed applications. The HTTP URL in the example is intended for local development only.

## Supported Struct Field Types

The loader automatically converts Yily variables into the following Go types:
- `string`
- `int`
- `int64`
- `time.Duration` (parsed via `time.ParseDuration`, e.g., `"5s"`, `"250ms"`, `"1h30m"`)
- `float64`
- `bool`

Fields tagged with `yily:"-"` or without any `yily` tag are ignored.

## Secret Refreshing

Applications can refresh secrets during runtime by recalling `Load` or `GetAll`:

- **Updating a struct**: Call `client.Load(ctx, &cfg)` again to retrieve the latest values. If retrieval or conversion fails for any field, the struct remains completely unchanged.
- **Fetching all variables**: Call `client.GetAll(ctx)` to obtain a fresh `map[string]string` containing all active environment variables. Note that because `GetAll` issues two sequential requests (listing variables, then reading their values), it is not an atomic snapshot if variables are modified or removed concurrently.

> **Note on `Watch`**: A real-time secret observation mechanism (`Watch`) for push updates or automated rotation without polling is being evaluated for future releases once the operational requirement is confirmed.

## Error Handling

The SDK exposes sentinel errors that can be inspected with standard `errors.Is`:
- `yily.ErrInvalidBaseURL`: the instance URL is malformed or uses a scheme other than HTTP or HTTPS.
- `yily.ErrInvalidToken`: the configured token does not have the Yily PAT prefix.
- `yily.ErrVariableNotFound`: a requested variable does not exist.
- `yily.ErrUnauthorized`: invalid or expired token (HTTP 401).
- `yily.ErrNotFound`: environment or resource not found (HTTP 404).
- `yily.ErrUnsupportedType`: struct field has an unsupported Go type.
- `yily.ErrInvalidTarget`: `Load` was called with an invalid target (must be a non-nil pointer to a struct).
- `yily.ErrInvalidValue`: a value could not be converted to the requested Go type. The error does not include the value.
- `yily.ErrInvalidVariableName`: `Get` was called with an empty variable name after normalization.
- `yily.ErrInvalidVersion`: a requested version is not a positive integer.
- `yily.ErrInvalidTag`: a struct tag is malformed or contains an unsupported option.
- `yily.ErrDuplicateVariable`: one struct requests different versions of the same Yily variable.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
