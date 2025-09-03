package models

//go:generate easyjson -all

// Types defined are struct aligned to conserve as much
// space as possible.
type Trade struct {
	// NOTE: Flaw of price is using float64 which creates
	// inexact precision when doing floating point ops.
	//
	// The ideal solution is to use a small abritrary precision
	// decimal library. There are some good competitors to shopsring decimal
	// these days.
	Size      float64 `json:"size"`
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
	TradeID   string  `json:"trade_id"`
	Symbol    string  `json:"symbol"`
}

//easyjson:json
type TradeList []Trade

// Candle represents an OHLC candle containing summary
// data about all trades occuring within a window of time
type Candle struct {
	Open float64 `json:"open"`
	High float64 `json:"high"`
	Low  float64 `json:"low"`
	// Close must be separate because not
	// All candles are successive
	Close float64 `json:"close"`
	// NOTE: Similar to the Price in Trade
	// Volume is a cumulative value of all trades
	// for the candle and it would be ideal to
	// use a arbitrary precision decimal library.
	//
	// Volume is in Quote/Notional.
	Volume float64 `json:"volume"`
	// Timestamp is the close time(ms) of the candle
	// This makes it easy to check if the candle
	// should be updated or not based on the current time
	Timestamp int64 `json:"timestamp"`
}

//easyjson:json
type CandleList []Candle
