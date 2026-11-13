# Go SDK Improvements

This document summarizes the improvements made to the Yily Go SDK for applications that consume configuration values and secrets.

## Load all variables

The client now provides:

```go
func (c *Client) GetAll(ctx context.Context) (map[string]string, error)
```

`GetAll` returns the active values in the client's environment as a map keyed by variable name. It first lists the variables, then fetches their values in one batch request. Changes made between these requests may affect the result, so the response is not an atomic snapshot.

The implementation reuses `fetchBatchValues`, which is also used by `Get` and `Load`. An empty environment returns an empty, non-nil map after the list request, without making a second request.

## Support `time.Duration`

`Load` now recognizes `time.Duration` and parses its value with `time.ParseDuration`. Values such as `5s`, `250ms`, `1h30m`, and `0` are supported. Invalid values return an error matching `ErrInvalidValue`; the original struct values remain unchanged if loading fails.

The README and example now document this type.

## Refresh values at runtime

Applications can call `Load` or `GetAll` again to retrieve the latest active values. The SDK does not start background refreshes automatically. A `Watch` API is deferred until an application requires automatic rotation without an explicit refresh call.

## Files updated

- `client.go`: added `GetAll` and reused the batch retrieval helper in `Get`.
- `loader.go`: added `fetchBatchValues` for shared batch retrieval.
- `loader.go`: added `time.Duration` conversion.
- `README.md`: documented `GetAll`, duration parsing, and runtime refresh.
- `examples/main.go`: added examples of `GetAll` and `time.Duration`.