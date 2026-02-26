package models

// PriceUpdateMessage is published to SNS every ~5 seconds during market hours.
// The notification Lambda subscribes to evaluate alert rules against live prices.
type PriceUpdateMessage struct {
	Timestamp int64                  `json:"timestamp"`
	Source    string                 `json:"source"` // "polygon_snapshot"
	Symbols   map[string]SymbolQuote `json:"symbols"`
}

// SymbolQuote is a lightweight price snapshot for a single symbol,
// used inside PriceUpdateMessage for SNS delivery.
type SymbolQuote struct {
	Price     float64 `json:"price"`
	Volume    int64   `json:"volume"`
	ChangePct float64 `json:"change_pct"`
}
