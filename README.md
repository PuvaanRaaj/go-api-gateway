# Go API Gateway – Phase 1

* Basic reverse proxy (path‑based routing)
* Request/response logging with request IDs
* Dockerfile + Docker Compose
* Two mock backend services (also in Go)
* Makefile for common tasks

> Default routes:
>
> * `http://localhost:8080/a/...` → Service A
> * `http://localhost:8080/b/...` → Service B

```


           +-----------------------------+
           |     Go API Gateway          |
           |        (Docker)             |
           +--------------+--------------+
                          |
        -----------------------------------------
        |                                       |
+---------------+                     +----------------+
|   Service A   |                     |   Service B    |
|   (Docker)    |                     |   (Docker)     |
+---------------+                     +----------------+

```


## 📁 Project Structure

```
api-gateway/
├─ cmd/
│  └─ gateway/
│     └─ main.go
├─ internal/
│  ├─ config/
│  │  └─ config.go
│  ├─ middleware/
│  │  └─ logging.go
│  ├─ proxy/
│  │  └─ proxy.go
│  └─ health/
│     └─ health.go
├─ mock/
│  ├─ service-a/
│  │  ├─ main.go
│  │  └─ Dockerfile
│  └─ service-b/
│     ├─ main.go
│     └─ Dockerfile
├─ pkg/
│  └─ version/
│     └─ version.go
├─ .env.example
├─ Dockerfile
├─ docker-compose.yml
├─ go.mod
├─ go.sum (generated)
├─ Makefile
└─ README.md
```

---

## 🔧 go.mod

```go
module github.com/yourname/api-gateway

go 1.22

require (
)
```

> Uses only the Go standard library.

---

## 🏁 cmd/gateway/main.go

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/yourname/api-gateway/internal/config"
    "github.com/yourname/api-gateway/internal/health"
    "github.com/yourname/api-gateway/internal/middleware"
    "github.com/yourname/api-gateway/internal/proxy"
)

func main() {
    cfg := config.Load()

    mux := http.NewServeMux()

    // Health
    mux.Handle("/healthz", health.Handler())

    // Reverse proxy routes
    mux.Handle("/a/", proxy.PathPrefixProxy("/a", cfg.BackendA))
    mux.Handle("/b/", proxy.PathPrefixProxy("/b", cfg.BackendB))

    // Root
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"name":"api-gateway","status":"ok"}`))
    })

    // Wrap with logging + request ID middleware
    handler := middleware.RequestID(middleware.Logger(mux))

    srv := &http.Server{
        Addr:              fmt.Sprintf(":%d", cfg.Port),
        Handler:           handler,
        ReadTimeout:       15 * time.Second,
        ReadHeaderTimeout: 10 * time.Second,
        WriteTimeout:      30 * time.Second,
        IdleTimeout:       60 * time.Second,
    }

    // Graceful shutdown
    go func() {
        log.Printf("gateway listening on :%d", cfg.Port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
    <-stop

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Printf("graceful shutdown failed: %v", err)
    }
    log.Printf("shutdown complete")
}
```

---

## ⚙️ internal/config/config.go

```go
package config

import (
    "log"
    "net/url"
    "os"
    "strconv"
)

type Config struct {
    Port     int
    BackendA *url.URL
    BackendB *url.URL
}

func mustParseURL(raw, fallback string) *url.URL {
    if raw == "" {
        raw = fallback
    }
    u, err := url.Parse(raw)
    if err != nil {
        log.Fatalf("invalid URL %q: %v", raw, err)
    }
    return u
}

func Load() *Config {
    port := 8080
    if p := os.Getenv("GATEWAY_PORT"); p != "" {
        if v, err := strconv.Atoi(p); err == nil {
            port = v
        }
    }

    a := mustParseURL(os.Getenv("BACKEND_A_URL"), "http://service-a:8080")
    b := mustParseURL(os.Getenv("BACKEND_B_URL"), "http://service-b:8080")

    return &Config{Port: port, BackendA: a, BackendB: b}
}
```

---

## 🔌 internal/proxy/proxy.go

```go
package proxy

import (
    "log"
    "net/http"
    "net/http/httputil"
    "net/url"
    "strings"
)

// PathPrefixProxy proxies requests whose path starts with prefix to target.
// It trims the prefix when forwarding so /a/foo → /foo on the backend.
func PathPrefixProxy(prefix string, target *url.URL) http.Handler {
    // Ensure prefix starts with '/'
    if !strings.HasPrefix(prefix, "/") {
        prefix = "/" + prefix
    }

    rp := httputil.NewSingleHostReverseProxy(target)

    // Customize the director to rewrite URL path and preserve query string.
    originalDirector := rp.Director
    rp.Director = func(r *http.Request) {
        originalDirector(r)
        // Rewrite scheme/host to target
        r.URL.Scheme = target.Scheme
        r.URL.Host = target.Host

        // Strip prefix (single leading instance)
        r.URL.Path = singlePrefixTrim(r.URL.Path, prefix)

        // Propagate X-Request-ID if present
        if rid := r.Context().Value(ContextKeyRequestID); rid != nil {
            r.Header.Set("X-Request-ID", rid.(string))
        }
    }

    // Optional: observe backend errors
    rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
        log.Printf("proxy error: %v", err)
        http.Error(w, "upstream unavailable", http.StatusBadGateway)
    }

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Only handle matching prefix
        if !strings.HasPrefix(r.URL.Path, prefix+"/") && r.URL.Path != prefix {
            http.NotFound(w, r)
            return
        }
        rp.ServeHTTP(w, r)
    })
}

// singlePrefixTrim removes the first occurrence of prefix from the path.
func singlePrefixTrim(path, prefix string) string {
    if path == prefix {
        return "/"
    }
    return strings.TrimPrefix(path, prefix)
}

// ContextKeyRequestID is set by middleware.RequestID.
type ContextKey string

const ContextKeyRequestID ContextKey = "request-id"
```

---

## 🧱 internal/middleware/logging.go

```go
package middleware

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "log"
    "net/http"
    "time"

    p "github.com/yourname/api-gateway/internal/proxy"
)

type statusRecorder struct {
    http.ResponseWriter
    status int
    size   int
}

func (w *statusRecorder) WriteHeader(code int) {
    w.status = code
    w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
    if w.status == 0 {
        w.status = http.StatusOK
    }
    n, err := w.ResponseWriter.Write(b)
    w.size += n
    return n, err
}

// Logger logs incoming requests & responses (status, size, duration, request ID).
func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        rid, _ := r.Context().Value(p.ContextKeyRequestID).(string)
        rec := &statusRecorder{ResponseWriter: w}

        next.ServeHTTP(rec, r)

        dur := time.Since(start)
        log.Printf("rid=%s method=%s path=%s status=%d size=%dB dur=%s remote=%s", rid, r.Method, r.URL.Path, rec.status, rec.size, dur, r.RemoteAddr)
    })
}

// RequestID injects a request ID into the context and response headers.
func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rid := genID()
        ctx := context.WithValue(r.Context(), p.ContextKeyRequestID, rid)
        w.Header().Set("X-Request-ID", rid)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func genID() string {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        // Very unlikely; fallback to timestamp-based
        return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
    }
    return hex.EncodeToString(b)
}
```

---

## ❤️ internal/health/health.go

```go
package health

import "net/http"

func Handler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"status":"ok"}`))
    })
}
```

---

## 🪪 pkg/version/version.go (optional)

```go
package version

var (
    Version = "dev"
    Commit  = ""
    BuiltAt = ""
)
```

> You can inject `-ldflags` during build later if desired.

---

## 🧪 Mock Service A – mock/service-a/main.go

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
)

type resp struct {
    Service    string            `json:"service"`
    Message    string            `json:"message"`
    RequestID  string            `json:"request_id"`
    RequestURI string            `json:"request_uri"`
    Headers    map[string]string `json:"headers"`
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        h := map[string]string{}
        for k, v := range r.Header {
            if len(v) > 0 {
                h[k] = v[0]
            }
        }
        out := resp{
            Service:    "service-a",
            Message:    "hello from service A",
            RequestID:  r.Header.Get("X-Request-ID"),
            RequestURI: r.RequestURI,
            Headers:    h,
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(out)
    })

    addr := ":8080"
    if p := os.Getenv("PORT"); p != "" {
        addr = ":" + p
    }
    log.Printf("service-a listening on %s", addr)
    log.Fatal(http.ListenAndServe(addr, mux))
}
```

### 🧪 Mock Service B – mock/service-b/main.go

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
)

type resp struct {
    Service    string `json:"service"`
    Message    string `json:"message"`
    RequestID  string `json:"request_id"`
    RequestURI string `json:"request_uri"`
}

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        out := resp{
            Service:    "service-b",
            Message:    "hello from service B",
            RequestID:  r.Header.Get("X-Request-ID"),
            RequestURI: r.RequestURI,
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(out)
    })

    addr := ":8080"
    if p := os.Getenv("PORT"); p != "" {
        addr = ":" + p
    }
    log.Printf("service-b listening on %s", addr)
    log.Fatal(http.ListenAndServe(addr, mux))
}
```

---

## 🐳 Gateway Dockerfile (multi‑stage)

```dockerfile
# -------- Build stage --------
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/gateway ./cmd/gateway

# -------- Runtime stage --------
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/gateway /usr/local/bin/gateway
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gateway"]
```

> Uses a minimal Distroless image for security.

---

## 🐳 Mock Service Dockerfiles

### mock/service-a/Dockerfile

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/service-a

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/service-a /usr/local/bin/service-a
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/service-a"]
```

### mock/service-b/Dockerfile

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/service-b

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/service-b /usr/local/bin/service-b
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/service-b"]
```

---

## 🧩 docker-compose.yml

```yaml
version: "3.9"
services:
  gateway:
    build: .
    image: api-gateway:dev
    environment:
      - GATEWAY_PORT=8080
      - BACKEND_A_URL=http://service-a:8080
      - BACKEND_B_URL=http://service-b:8080
    ports:
      - "8080:8080"
    depends_on:
      - service-a
      - service-b

  service-a:
    build: ./mock/service-a
    image: mock-service-a:dev

  service-b:
    build: ./mock/service-b
    image: mock-service-b:dev
```

---

## 📄 .env.example

```
# Gateway
GATEWAY_PORT=8080
BACKEND_A_URL=http://service-a:8080
BACKEND_B_URL=http://service-b:8080
```

> docker-compose already wires sensible defaults; override as needed.

---

## 🧰 Makefile

```makefile
APP_NAME=api-gateway
BIN=./bin/gateway
PKG=github.com/yourname/api-gateway

.PHONY: all build run test clean docker-build up down logs fmt vet tidy

all: build

build:
	GOFLAGS=-trimpath CGO_ENABLED=0 go build -o $(BIN) ./cmd/gateway

run:
	go run ./cmd/gateway

test:
	go test ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin dist

# Docker / Compose

docker-build:
	docker build -t $(APP_NAME):dev .

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f gateway
```

---

## 📘 README.md

````markdown
# API Gateway (Phase 1)

Minimal Go reverse proxy with logging and Dockerized mock services.

## Quickstart

```bash
make up
# then
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/a/hello
curl -s http://localhost:8080/b/hello
````

Check logs:

```bash
make logs
```

## Configuration

* `GATEWAY_PORT` (default `8080`)
* `BACKEND_A_URL` (default `http://service-a:8080`)
* `BACKEND_B_URL` (default `http://service-b:8080`)

## Notes

* Path prefix `/a` and `/b` are stripped before forwarding.
* `X-Request-ID` is added at the gateway and propagated downstream.
* Sensible server timeouts + graceful shutdown included.

```

---

## ✅ Sanity Checks

After `make up`:

- `GET /healthz` → `{ "status": "ok" }`
- `GET /a/hello` → JSON from Service A (with `request_id`)
- `GET /b/hello` → JSON from Service B (with `request_id`)
- Gateway logs show `method, path, status, size, duration, remote, rid`

---

## Next Steps (Phase 2 ideas)

- Env‑driven dynamic route table (JSON/YAML)
- Circuit breaking / retries / timeouts per route
- Metrics (Prometheus) + structured logs (slog/zap)
- AuthN/Z middlewares (JWT, API keys, HMAC)
- Rate limiting / quota per client
- OpenAPI docs for admin endpoints
```
