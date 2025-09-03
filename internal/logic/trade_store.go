package logic

import (
	"sync"

	"github.com/infinityCounter2/vh-trader/internal/models"
)

type TradeStoreParams struct {
	// CacheLimit is the maximum number
	// of trades to keep for each symbol.
	//
	// Defaults to 50.
	CacheLimit int
}

// TradeStore is a cache of trade events
// that have been pushed for all symbols.
//
// The trade stores up to the limit specified
// by the CacheLimit argument for each symbol.
type TradeStore struct {
	p   TradeStoreParams
	mtx sync.RWMutex
	// Keyed by Symbol
	trades map[string][]models.Trade
}

func NewTradeStore(p TradeStoreParams) *TradeStore {
	if p.CacheLimit <= 0 {
		p.CacheLimit = 50
	}

	return &TradeStore{
		p:      p,
		trades: make(map[string][]models.Trade),
		mtx:    sync.RWMutex{},
	}
}

// PushTrades stores new trades.
func (store *TradeStore) PushTrades(trades []models.Trade) {
	store.mtx.Lock()
	defer store.mtx.Unlock()

	for _, trade := range trades {
		symbolTrades := store.trades[trade.Symbol]
		if symbolTrades == nil {
			symbolTrades = make([]models.Trade, 0, store.p.CacheLimit)
		}

		// If cache is full and new trade is older than or equal to the oldest cached trade, skip.
		if len(symbolTrades) == store.p.CacheLimit && trade.Timestamp <= symbolTrades[0].Timestamp {
			continue // This new trade is too old to be in the cache
		}

		// Find the correct position to insert the new trade to maintain sorted order (oldest to front, newest to back)
		inserted := false
		for i := 0; i < len(symbolTrades); i++ {
			if trade.Timestamp < symbolTrades[i].Timestamp {
				// Insert trade before symbolTrades[i]
				symbolTrades = append(symbolTrades[:i], append([]models.Trade{trade}, symbolTrades[i:]...)...)
				inserted = true
				break
			}
		}

		if !inserted {
			// If newTrade is the newest or the slice was empty, append it to the end
			symbolTrades = append(symbolTrades, trade)
		}

		// Trim the slice to the CacheLimit if it exceeds it, keeping the newest trades at the end.
		if len(symbolTrades) > store.p.CacheLimit {
			symbolTrades = symbolTrades[len(symbolTrades)-store.p.CacheLimit:]
		}

		store.trades[trade.Symbol] = symbolTrades
	}
}

// GetTrades retrieves the cached trades for a given symbol.
// It returns a slice of trades, sorted from oldest to newest.
func (store *TradeStore) GetTrades(symbol string) []models.Trade {
	store.mtx.RLock()
	defer store.mtx.RUnlock()

	symbolTrades, exists := store.trades[symbol]
	if !exists || len(symbolTrades) == 0 {
		return []models.Trade{}
	}

	// Return a copy to prevent external modification of the cached slice.
	tradesCopy := make([]models.Trade, len(symbolTrades))
	copy(tradesCopy, symbolTrades)
	return tradesCopy
}
