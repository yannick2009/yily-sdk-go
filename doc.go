// Package yily is a lightweight client for reading configuration variables
// and secrets from a Yily instance.
//
// Create a client with NewClient, then choose the retrieval method that fits
// the use case:
//
//   - Load fills a struct from exported fields tagged with `yily`.
//   - Get reads one known variable.
//   - GetAll reads every variable of the environment as a map.
//
// Example:
//
//	type AppConfig struct {
//		Host string `yily:"DB_HOST"`
//		Port int    `yily:"DB_PORT"`
//	}
//
//	client, err := yily.NewClient(yily.Config{
//		BaseURL:       "https://yily.example.com",
//		EnvironmentID: "your-environment-uuid",
//		Token:         os.Getenv("YILY_TOKEN"),
//	})
//	if err != nil {
//		return err
//	}
//
//	var cfg AppConfig
//	if err := client.Load(ctx, &cfg); err != nil {
//		return err
//	}
//
// # File map
//
//   - client.go: Client type, public methods, and HTTP internals.
//   - config.go: Config type and its validation.
//   - loader.go: struct inspection and value decoding used by Load.
//   - errors.go: sentinel errors and APIError.
//   - utils.go: shared variable-name and base-URL normalization.
package yily
