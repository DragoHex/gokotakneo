# gokotakneo

[![Go Reference](https://pkg.go.dev/badge/github.com/DragoHex/gokotakneo.svg)](https://pkg.go.dev/github.com/DragoHex/gokotakneo)
[![Go Report Card](https://goreportcard.com/badge/github.com/DragoHex/gokotakneo)](https://goreportcard.com/report/github.com/DragoHex/gokotakneo)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Production-grade, high-throughput Go SDK for the **Kotak Neo Trade API** gateway. Built with 1:1 functional parity in the image of `gokiteconnect`, featuring two-step TOTP/MPIN authentication with dynamic gateway discovery, high-performance binary batch tick streaming with price divider scaling, real-time order updates, and thread-safe REST execution.

---

## Architecture & Package Layout

`gokotakneo` is structured into clean domain subpackages:

```
gokotakneo/
├── client.go              # Root Client & session gateway state
├── proxy.go               # Static forward proxy support (WithProxy / ProxyConfig / RESTClientFor)
├── transport.go           # JData HTTPClient pipeline & session header injection
├── endpoints.go           # Kotak Neo REST URI route constants
├── errors.go              # Structured APIError and domain sentinels
├── auth.go                # 2-step TOTP login, MPIN validation & dynamic routing
├── orders.go              # Regular order execution, modification & cancellation
├── portfolio.go           # Positions, Demat Holdings & Margin conversions
├── market.go              # Market Quotes, LTP & Depth snapshots
├── reports.go             # Order book, trade book & margin limit queries
│
├── models/                # Domain data models & custom serialization
│   ├── models.go          # OrderParams, OrderResponse, Position, Limit
│   └── time.go            # Custom timestamp deserializer (models.Time)
│
├── feed/                  # Real-Time Market Quotes Binary WebSocket
│   ├── feed.go            # FeedClient connection loop & watchdog
│   ├── parser.go          # Batch binary tick parser with price divider scaling
│   └── types.go           # Tick, Depth, FeedCallbacks, Subscription
│
├── orderstream/           # Real-Time Order & Position Event WebSocket
│   ├── stream.go          # Interactive socket connection loop & message demux
│   └── types.go           # OrderUpdate, StreamCallbacks
│
└── example/
    └── main.go            # End-to-end runnable demonstration
```

---

## Key Features

- **Standardized Namespace**: Packaged as `github.com/DragoHex/gokotakneo`.
- **Two-Step Dynamic Authentication**:
  - Step 1: Login with TOTP to receive view token.
  - Step 2: Validate MPIN to receive trade session and dynamically discover cluster gateways (`baseURL`, `feedURL`, `rtURL`).
- **Binary Batch Streaming (`feed`)**:
  - Parses Kotak Neo binary tick frames with high efficiency.
  - Applies automated price divider scaling across segments.
- **Order Lifecycle Stream (`orderstream`)**: Real-time push notifications for order fills, modifications, and rejections.
- **Context First**: Every blocking network call accepts `context.Context` for proper cancellation and deadline management.
- **Strict Error Handling**: Typed domain sentinels (`ErrUnauthorized`, `ErrOrderNotFound`, `ErrBrokerUnreachable`).

---

## Installation

```sh
go get github.com/DragoHex/gokotakneo
```

Requires Go `1.23` or later.

---

## Quickstart

### 1. Two-Step Authentication & Gateway Discovery

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DragoHex/gokotakneo"
)

func main() {
	ctx := context.Background()

	// Initialize with Neo Developer Portal Access Token
	client := gokotakneo.New("your-portal-access-token")

	// Step 1: Login with TOTP
	viewSession, err := client.LoginTOTP(ctx, "+919876543210", "UCC123", "123456")
	if err != nil {
		log.Fatalf("LoginTOTP failed: %v", err)
	}
	fmt.Printf("[+] Step 1 OK. View Token: %s...\n", viewSession.Token[:10])

	// Step 2: Validate MPIN (Discovers dynamic baseURL, feedURL, rtURL)
	tradeSession, err := client.ValidateMPIN(ctx, viewSession.Token, viewSession.Sid, "654321")
	if err != nil {
		log.Fatalf("ValidateMPIN failed: %v", err)
	}

	fmt.Printf("[+] Step 2 OK. Gateways Discovered:\n  REST Base: %s\n  Quotes Feed: %s\n  Order Stream: %s\n",
		client.BaseURL(), client.FeedURL(), client.RTURL())
}
```

---

### 2. Place & Audit Orders

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DragoHex/gokotakneo"
	"github.com/DragoHex/gokotakneo/models"
)

func main() {
	ctx := context.Background()
	client := gokotakneo.New("your-portal-access-token")

	// Place Limit Buy Order
	orderResp, err := client.PlaceOrder(ctx, models.OrderParams{
		Am: "NO",       // After Market Order: NO
		Dq: "0",        // Disclosed Quantity: 0
		Es: "nse_cm",   // Exchange Segment
		Mp: "0",        // Market Protection: 0
		Pc: "CNC",      // Product Code: CNC / MIS / NRML
		Pf: "N",        // Price Flag: Normal
		Pr: "1450.50",  // Price
		Pt: "L",        // Order Type: L (Limit) / MKT (Market) / SL / SL-M
		Qt: "10",       // Quantity
		Rt: "DAY",      // Retention Type: DAY / IOC
		Tp: "0",        // Trigger Price
		Ts: "INFY-EQ",  // Trading Symbol
		Tt: "B",        // Transaction Type: B (Buy) / S (Sell)
	})
	if err != nil {
		log.Fatalf("PlaceOrder error: %v", err)
	}

	fmt.Printf("Order Submitted! Order Number: %s\n", orderResp.OrderNumber)

	// Fetch Order Book
	orders, err := client.GetOrderBook(ctx)
	if err != nil {
		log.Fatalf("GetOrderBook error: %v", err)
	}
	for _, o := range orders {
		fmt.Printf("Order %s: %s %s @ %s [%s]\n",
			o.OrderNumber, o.TradingSymbol, o.Quantity, o.Price, o.Status)
	}
}
```

---

### 3. Live Market Quotes Binary Feed (`feed`)

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DragoHex/gokotakneo/feed"
)

func main() {
	feedCallbacks := feed.FeedCallbacks{
		OnConnect: func() {
			fmt.Println("[WebSocket] Connected to Kotak Neo Market Feed!")
		},
		OnTick: func(tick feed.Tick) {
			fmt.Printf("[Tick] Token: %d, LTP: %s, Volume: %d\n",
				tick.InstrumentToken, tick.LastPrice, tick.Volume)
		},
		OnError: func(err error) {
			fmt.Printf("[WebSocket Error] %v\n", err)
		},
	}

	feedClient := feed.NewClient(
		"wss://feed.kotaksecurities.com",
		"your-ucc",
		"your-session-sid",
		feedCallbacks,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	feedClient.Serve(ctx)
}
```

---

### 4. Real-time Order Updates Stream (`orderstream`)

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/DragoHex/gokotakneo/orderstream"
)

func main() {
	orderCallbacks := orderstream.StreamCallbacks{
		OnConnect: func() {
			fmt.Println("[WebSocket] Connected to Kotak Neo Order Stream!")
		},
		OnOrderUpdate: func(order orderstream.OrderUpdate) {
			fmt.Printf("[Order Alert] %s -> Status: %s, FilledQty: %s\n",
				order.OrderNumber, order.Status, order.FilledQuantity)
		},
	}

	streamClient := orderstream.New(
		"wss://rt.kotaksecurities.com",
		"your-token",
		"your-session-sid",
		orderCallbacks,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	streamClient.Serve(ctx)
}
```

---

### 5. Static IP & Forward Proxy (Algo IP Whitelisting)

Configure dedicated forward proxies (such as `algoip.in`, `staticip.in`, or private forward proxies) for Kotak Neo IP whitelisting:

```go
// Option A: Structured ProxyConfig
client := gokotakneo.NewClient("consumer-key",
	gokotakneo.WithProxy(gokotakneo.ProxyConfig{
		Host:         "dc46-mum-01.algoip.in",
		Port:         443,
		ClientID:     "your_proxy_username",
		ClientSecret: "your_proxy_password",
	}),
)

// Option B: Proxy URL string
client := gokotakneo.NewClient("consumer-key",
	gokotakneo.WithProxyURL("https://user:pass@dc46-mum-01.algoip.in:443"),
)

// Option C: Standalone RESTClientFor
httpClient, err := gokotakneo.RESTClientFor(gokotakneo.ProxyConfig{
	Host:         "dc46-mum-01.algoip.in",
	Port:         443,
	ClientID:     "user",
	ClientSecret: "pass",
})

// WebSocket Feed and OrderStream through proxy
feedClient := feed.New("wss://rt.kotaksecurities.com", "token", "sid", callbacks)
feedClient.SetProxyURL("https://user:pass@dc46-mum-01.algoip.in:443")

streamClient := orderstream.New("wss://rt.kotaksecurities.com", "token", "sid", orderCallbacks)
streamClient.SetProxyURL("https://user:pass@dc46-mum-01.algoip.in:443")
```

---

## Error Handling & Sentinels

```go
if err != nil {
	switch {
	case errors.Is(err, gokotakneo.ErrUnauthorized):
		// Token expired or invalid session — re-run MPIN / TOTP
	case errors.Is(err, gokotakneo.ErrRateLimit):
		// Neo gateway rate limit hit
	case errors.Is(err, gokotakneo.ErrNotFound):
		// Resource not found
	default:
		var apiErr *gokotakneo.APIError
		if errors.As(err, &apiErr) {
			fmt.Printf("Kotak Neo API Error [%s]: %s (HTTP %d)\n",
				apiErr.ErrorCode, apiErr.Message, apiErr.HTTPStatus)
		}
	}
}
```

---

## Testing & Verification

Run tests with race detection:

```sh
go test -race ./...
```

Run static analysis:

```sh
go vet ./...
```

Build the runnable example:

```sh
go build -o /dev/null ./example/main.go
```
