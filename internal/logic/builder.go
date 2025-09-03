package logic

import (
	"sort"
	"time"

	"github.com/infinityCounter2/vh-trader/internal/models"
)

// BuilderInterval is a candle size, acceptable values are '60', '300', '1500', '3600',
type BuilderInterval = time.Duration

const (
	BuilderInterval1m  BuilderInterval = time.Minute
	BuilderInterval5m  BuilderInterval = 5 * time.Minute
	BuilderInterval15m BuilderInterval = 15 * time.Minute
	BuilderInterval1h  BuilderInterval = time.Hour
)

var BuilderIntervals = []BuilderInterval{
	BuilderInterval1m,
	BuilderInterval5m,
	BuilderInterval15m,
	BuilderInterval1h,
}

type CandleBuilderParams struct {
	Interval BuilderInterval
}

// CandleBuilder is a structure that processes trades
// to consturct Candle candles
type CandleBuilder struct {
	p CandleBuilderParams

	current *models.Candle

	// closed contains all the Candle
	// Candles that have alread elapsed.
	//
	// These should be routinely flushed
	// to a in-memory kv store like redis
	// and to a more persistent store like postgres.
	// At that point a mutex should be added to protect this.
	//
	// Keyed by the timestamp of the candle.
	closed map[int64]models.Candle
}

func NewBuilder(p CandleBuilderParams) *CandleBuilder {
	return &CandleBuilder{
		p:      p,
		closed: make(map[int64]models.Candle),
	}
}

// ProcessTrades batch updates the CandleBuilder with the trades given.
//
// The appropriate Candle candle is updated for each trade included. Candles
// that do not exist at the time will be created.
func (c *CandleBuilder) ProcessTrades(trades []models.Trade) {
	for _, t := range trades {
		c.processTrade(t)
	}
}

func (c *CandleBuilder) processTrade(t models.Trade) {
	exec := time.Unix(0, t.Timestamp*int64(time.Millisecond)).UTC()
	tradeCandleTime := roundUpTime(exec, c.p.Interval)

	if c.current != nil && tradeCandleTime == c.current.Timestamp {
		// This trade belongs in this candle.
		updateCandle(c.current, t)
	} else if c.current == nil || (c.current != nil && tradeCandleTime > c.current.Timestamp) {
		// This is a new candle
		candle := initializeCandle(tradeCandleTime, t)
		if c.current != nil {
			c.closed[c.current.Timestamp] = *c.current
		}
		c.current = candle
	} else if tradeCandleTime < c.current.Timestamp {
		// This is an old trade, really we should have a deep discussion
		// on how to handle late trades before doing this but just update the old candle
		// it may belong to.
		candle, exists := c.closed[tradeCandleTime]
		if exists {
			updateCandle(&candle, t)
		} else {
			// Build a new candle
			candle = *(initializeCandle(tradeCandleTime, t))
		}
		c.closed[tradeCandleTime] = candle
	}
}

// GetCandles returns all the candles in the builder including the closed ones.
// Ideally the builder has some POP Candles method that is called by client code
// to fetch closed candles, and then that's used by some API/data layer instead
// of having this GetCandles method.
func (c *CandleBuilder) GetCandles() models.CandleList {
	if c.current == nil && len(c.closed) == 0 {
		return nil
	}

	candles := make([]models.Candle, 0, len(c.closed)+1)
	for _, c := range c.closed {
		candles = append(candles, c)
	}

	candles = append(candles, *c.current)

	// Sort candles in Chronological order.
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].Timestamp < candles[j].Timestamp
	})

	return models.CandleList(candles)
}

func initializeCandle(candleTime int64, t models.Trade) *models.Candle {
	candle := &models.Candle{
		Timestamp: candleTime,
	}
	updateCandle(candle, t)
	return candle
}

func updateCandle(candle *models.Candle, t models.Trade) {
	if candle.Volume == 0 {
		// Empty candle so use this trade to set all values
		candle.High = t.Price
		candle.Low = t.Price
		candle.Open = t.Price
	} else if t.Price > candle.High {
		candle.High = t.Price
	} else if t.Price < candle.Low {
		candle.Low = t.Price
	}
	// NOTE: Use a arbitray precision float point library here like decimal.Decimal as we
	// may run into issues numbers exceeding the capacity of float64
	candle.Volume += t.Size * t.Price
	candle.Close = t.Price
}

// roundUpTime rounds the given time to the next multiple of the duration.
// Use to calculate close time of candle for a trade.
func roundUpTime(t time.Time, d time.Duration) int64 {
	return t.Truncate(d).Add(d).Unix() * 1000
}
