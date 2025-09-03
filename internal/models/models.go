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
	Price     float64 // 8 bytes
	Timestamp int64   // 8 bytes
	TradeID   string  // string header (16 bytes, pointer+len)
	Symbol    string  // string header (16 bytes)
	Size      float32 // 4 bytes
}

// OHLC represents a candle containing summary
// data about all trades occuring within a window of time
type OHLC struct {
	Open float64
	High float64
	Low  float64
	// Close must be separate because not
	// All candles are successive
	Close float64
	// NOTE: Similar to the Price in Trade
	// Volume is a cumulative value of all trades
	// for the candle and it would be ideal to
	// use a arbitrary precision decimal library.
	Volume    float64
	Timestamp int64
}
