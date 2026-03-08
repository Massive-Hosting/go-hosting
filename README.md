# go-hosting

Go client library for the [Hosting Platform](https://massive-hosting.com) control panel API.

## Installation

```bash
go get github.com/massive-hosting/go-hosting
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/massive-hosting/go-hosting"
)

type Webapp struct {
    ID     string `json:"id"`
    Status string `json:"status"`
}

func main() {
    client := hosting.New("https://home.massive-hosting.com", "hst_pat_your_token_here")
    ctx := context.Background()

    // Create a resource
    webapp, err := hosting.Post[Webapp](ctx, client, "/api/v1/customers/cust123/webapps", map[string]any{
        "tenant_id":       "t_abc123",
        "runtime":         "php",
        "runtime_version": "8.4",
    })
    if err != nil {
        panic(err)
    }
    fmt.Println("Created:", webapp.ID)

    // Get a resource
    webapp, err = hosting.Get[Webapp](ctx, client, "/api/v1/webapps/"+webapp.ID)
    if err != nil {
        panic(err)
    }
    fmt.Println("Status:", webapp.Status)

    // List resources
    webapps, err := hosting.List[Webapp](ctx, client, "/api/v1/customers/cust123/webapps")
    if err != nil {
        panic(err)
    }
    fmt.Println("Count:", len(webapps))

    // Delete a resource
    err = client.Delete(ctx, "/api/v1/webapps/"+webapp.ID)
    if err != nil {
        panic(err)
    }

    // Wait for async provisioning
    webapp, err = hosting.WaitForStatus[Webapp](ctx, client, "/api/v1/webapps/"+webapp.ID,
        func(w *Webapp) string { return w.Status }, 5*time.Minute)
    if err != nil {
        panic(err)
    }
}
```

## Authentication

The client authenticates using Personal Access Tokens (PATs). Create one from your profile page in the control panel.

## API

| Function | Description |
|---|---|
| `hosting.New(baseURL, token)` | Create a new client |
| `hosting.Get[T](ctx, client, path)` | GET and decode JSON response |
| `hosting.List[T](ctx, client, path)` | GET and unwrap list envelope (`{items: [...]}`) |
| `hosting.Post[T](ctx, client, path, body)` | POST with JSON body |
| `hosting.Put[T](ctx, client, path, body)` | PUT with JSON body |
| `hosting.Patch[T](ctx, client, path, body)` | PATCH with JSON body |
| `client.Delete(ctx, path)` | DELETE (no response body) |
| `hosting.WaitForStatus[T](...)` | Poll until resource reaches `active`/`running` or fails |
| `hosting.IsNotFound(err)` | Check if error is a 404 |
| `hosting.StatusCode(err)` | Extract HTTP status code from error |

## License

MIT
