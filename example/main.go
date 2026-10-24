package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DragoHex/gokotakneo"
	"github.com/DragoHex/gokotakneo/feed"
	"github.com/DragoHex/gokotakneo/models"
	"github.com/DragoHex/gokotakneo/orderstream"
)

func main() {
	ctx := context.Background()

	// 1. Initialize Client with NEO Developer Portal Access Token
	client := gokotakneo.New("your-portal-access-token")

	// 2. Step 1: Login with TOTP
	viewSession, err := client.LoginTOTP(ctx, "+919876543210", "UCC123", "123456")
	if err != nil {
		log.Fatalf("LoginTOTP failed: %v", err)
	}
	fmt.Printf("[+] Step 1 Success. View Token: %s...\n", viewSession.Token[:10])

	// 3. Step 2: Validate MPIN (Discovers dynamic baseUrl, feedUrl, rtUrl)
	tradeSession, err := client.ValidateMPIN(ctx, viewSession.Token, viewSession.Sid, "654321")
	if err != nil {
		log.Fatalf("ValidateMPIN failed: %v", err)
	}
	fmt.Printf("[+] Step 2 Success. Discovered Gateways:\n  Base: %s\n  Feed: %s\n  Realtime: %s\n",
		client.BaseURL(), client.FeedURL(), client.RTURL())

	// 4. Place a Limit Buy Order
	orderResp, err := client.PlaceOrder(ctx, models.OrderParams{
		Am: "NO",
		Dq: "0",
		Es: "nse_cm",
		Mp: "0",
		Pc: "CNC",
		Pf: "N",
		Pr: "1450.50",
		Pt: "L",
		Qt: "10",
		Rt: "DAY",
		Tp: "0",
		Ts: "INFY-EQ",
		Tt: "B",
	})
	if err != nil {
		log.Printf("PlaceOrder: %v", err)
	} else {
		fmt.Printf("[+] Order Placed! Number: %s\n", orderResp.OrderNumber)
	}

	// 5. Query Positions & Limits
	positions, err := client.GetPositions(ctx)
	if err == nil {
		fmt.Printf("[+] Open Positions: %d\n", len(positions))
	}

	limits, err := client.GetLimits(ctx, "ALL", "ALL", "ALL")
	if err == nil {
		fmt.Printf("[+] Available Net Limit: %s\n", limits.Net)
	}

	// 6. Connect to Live Market Quotes Feed
	feedCallbacks := feed.FeedCallbacks{
		OnConnect: func() {
			fmt.Println("[+] Connected to Market Quotes Feed!")
		},
		OnTick: func(tick feed.Tick) {
			fmt.Printf("[Tick] Token: %d, LTP: %s\n", tick.InstrumentToken, tick.LastPrice)
		},
	}
	feedClient := feed.NewClient(client.FeedURL(), tradeSession.UCC, tradeSession.Sid, feedCallbacks)
	feedCtx, feedCancel := context.WithTimeout(ctx, 5*time.Second)
	defer feedCancel()
	go feedClient.Serve(feedCtx)

	// 7. Connect to Order & Position Event Stream
	orderCallbacks := orderstream.StreamCallbacks{
		OnConnect: func() {
			fmt.Println("[+] Connected to Live Order Stream!")
		},
		OnOrderUpdate: func(order orderstream.OrderUpdate) {
			fmt.Printf("[Order Update] %s -> Status: %s\n", order.OrderNumber, order.Status)
		},
	}
	orderStream := orderstream.New(client.RTURL(), tradeSession.Token, tradeSession.Sid, orderCallbacks)
	orderCtx, orderCancel := context.WithTimeout(ctx, 5*time.Second)
	defer orderCancel()
	go orderStream.Serve(orderCtx)

	time.Sleep(100 * time.Millisecond)
}
