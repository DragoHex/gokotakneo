package gokotakneo

// API endpoints for Kotak-Neo trade APIs.
const (
	// Authentication
	URILoginTOTP    string = "/login/1.0/tradeApiLogin"
	URIValidateMPIN string = "/login/1.0/tradeApiValidate"

	// Orders
	URIPlaceOrder  string = "/quick/order/rule/ms/place"
	URIModifyOrder string = "/quick/order/vr/modify"
	URICancelOrder string = "/quick/order/cancel"

	// Reports
	URIGetOrders       string = "/quick/user/orders"
	URIGetOrderByID    string = "/quick/user/orders/%s"
	URIGetOrderHistory string = "/quick/user/order/history"
	URIGetTrades       string = "/quick/user/trades"

	// Portfolio & Balances
	URIGetPositions string = "/quick/user/positions"
	URIGetHoldings  string = "/portfolio/v1/holdings"
	URIGetLimits    string = "/quick/user/limits"

	// Market Data Quotes
	URIQuotes string = "/script-details/1.0/quotes/neosymbol/%s"
	URILTP    string = "/script-details/1.0/quotes/neosymbol/%s/ltp"

	// WebSockets
	URIFeedWS     string = "/apifeed"
	URIRealtimeWS string = "/realtime"
)
