package logic

import (
	"testing"
	"time"

	"github.com/infinityCounter2/vh-trader/internal/models"
	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	params := CandleBuilderParams{Interval: BuilderInterval1h}
	builder := NewBuilder(params)

	require.NotNil(t, builder, "NewBuilder returned nil")
	require.Equal(t, BuilderInterval1h, builder.p.Interval, "Interval mismatch")
	require.Nil(t, builder.current, "Expected current candle to be nil")
	require.NotNil(t, builder.closed, "Expected closed map to be initialized")
	require.Empty(t, builder.closed, "Expected closed map to be empty")
}

func TestProcessTrade_NewCandle(t *testing.T) {
	params := CandleBuilderParams{Interval: BuilderInterval1m}
	builder := NewBuilder(params)

	tradeTime := time.Date(2023, 1, 1, 10, 0, 30, 0, time.UTC)
	trade := models.Trade{
		TradeID:   "1",
		Timestamp: tradeTime.Unix(),
		Price:     100.0,
		Size:      1.0,
	}

	builder.processTrade(trade)

	require.NotNil(t, builder.current, "Expected current candle to be initialized")

	expectedCandleTime := roundUpTime(tradeTime, BuilderInterval1m)
	require.Equal(t, expectedCandleTime, builder.current.Timestamp, "Current candle timestamp mismatch")
	require.Equal(t, trade.Price, builder.current.Open, "Open price mismatch")
	require.Equal(t, trade.Price, builder.current.High, "High price mismatch")
	require.Equal(t, trade.Price, builder.current.Low, "Low price mismatch")
	require.Equal(t, trade.Price, builder.current.Close, "Close price mismatch")
	require.Equal(t, trade.Size*trade.Price, builder.current.Volume, "Volume mismatch")
	require.Empty(t, builder.closed, "Expected closed candles map to be empty")
}

func TestProcessTrade_SameCandle(t *testing.T) {
	params := CandleBuilderParams{Interval: BuilderInterval1m}
	builder := NewBuilder(params)

	tradeTime1 := time.Date(2023, 1, 1, 10, 0, 10, 0, time.UTC)
	trade1 := models.Trade{TradeID: "1", Timestamp: tradeTime1.Unix(), Price: 100.0, Size: 1.0}
	builder.processTrade(trade1)

	tradeTime2 := time.Date(2023, 1, 1, 10, 0, 20, 0, time.UTC)
	trade2 := models.Trade{TradeID: "2", Timestamp: tradeTime2.Unix(), Price: 105.0, Size: 2.0}
	builder.processTrade(trade2)

	tradeTime3 := time.Date(2023, 1, 1, 10, 0, 40, 0, time.UTC)
	trade3 := models.Trade{TradeID: "3", Timestamp: tradeTime3.Unix(), Price: 95.0, Size: 3.0}
	builder.processTrade(trade3)

	require.NotNil(t, builder.current, "Expected current candle to be initialized")

	expectedCandleTime := roundUpTime(tradeTime1, BuilderInterval1m)
	require.Equal(t, expectedCandleTime, builder.current.Timestamp, "Current candle timestamp mismatch")
	require.Equal(t, trade1.Price, builder.current.Open, "Open price mismatch")
	require.Equal(t, trade2.Price, builder.current.High, "High price mismatch")
	require.Equal(t, trade3.Price, builder.current.Low, "Low price mismatch")
	require.Equal(t, trade3.Price, builder.current.Close, "Close price mismatch")

	expectedVolume := (trade1.Size * trade1.Price) + (trade2.Size * trade2.Price) + (trade3.Size * trade3.Price)
	require.Equal(t, expectedVolume, builder.current.Volume, "Volume mismatch")
	require.Empty(t, builder.closed, "Expected closed candles map to be empty")
}

func TestProcessTrade_NewInterval(t *testing.T) {
	params := CandleBuilderParams{Interval: BuilderInterval1m}
	builder := NewBuilder(params)

	tradeTime1 := time.Date(2023, 1, 1, 10, 0, 10, 0, time.UTC)
	trade1 := models.Trade{TradeID: "1", Timestamp: tradeTime1.Unix(), Price: 100.0, Size: 1.0}
	builder.processTrade(trade1)

	tradeTime2 := time.Date(2023, 1, 1, 10, 1, 0, 0, time.UTC) // New minute
	trade2 := models.Trade{TradeID: "2", Timestamp: tradeTime2.Unix(), Price: 110.0, Size: 2.0}
	builder.processTrade(trade2)

	require.NotNil(t, builder.current, "Expected current candle to be initialized")

	// Check closed candle
	require.Len(t, builder.closed, 1, "Expected 1 closed candle")
	expectedClosedCandleTime := roundUpTime(tradeTime1, BuilderInterval1m)
	closedCandle, exists := builder.closed[expectedClosedCandleTime]
	require.True(t, exists, "Expected closed candle with timestamp %v not found", expectedClosedCandleTime)
	require.Equal(t, trade1.Price, closedCandle.Open, "Closed candle Open incorrect")
	require.Equal(t, trade1.Price, closedCandle.High, "Closed candle High incorrect")
	require.Equal(t, trade1.Price, closedCandle.Low, "Closed candle Low incorrect")
	require.Equal(t, trade1.Price, closedCandle.Close, "Closed candle Close incorrect")
	require.Equal(t, trade1.Size*trade1.Price, closedCandle.Volume, "Closed candle Volume incorrect")

	// Check current candle
	expectedCurrentCandleTime := roundUpTime(tradeTime2, BuilderInterval1m)
	require.Equal(t, expectedCurrentCandleTime, builder.current.Timestamp, "Current candle timestamp mismatch")
	require.Equal(t, trade2.Price, builder.current.Open, "Current candle Open incorrect")
	require.Equal(t, trade2.Price, builder.current.High, "Current candle High incorrect")
	require.Equal(t, trade2.Price, builder.current.Low, "Current candle Low incorrect")
	require.Equal(t, trade2.Price, builder.current.Close, "Current candle Close incorrect")
	require.Equal(t, trade2.Size*trade2.Price, builder.current.Volume, "Current candle Volume incorrect")
}

func TestProcessTrade_LateTrade(t *testing.T) {
	params := CandleBuilderParams{Interval: BuilderInterval1m}
	builder := NewBuilder(params)

	// First trade, creates a candle at 10:01
	tradeTime1 := time.Date(2023, 1, 1, 10, 0, 30, 0, time.UTC) // This will round up to 10:01
	trade1 := models.Trade{TradeID: "1", Timestamp: tradeTime1.Unix(), Price: 100.0, Size: 1.0}
	builder.processTrade(trade1)

	// Second trade, creates a candle at 10:02 and closes the 10:01 candle
	tradeTime2 := time.Date(2023, 1, 1, 10, 1, 30, 0, time.UTC) // This will round up to 10:02
	trade2 := models.Trade{TradeID: "2", Timestamp: tradeTime2.Unix(), Price: 120.0, Size: 2.0}
	builder.processTrade(trade2)

	// Late trade for the 10:01 candle
	lateTradeTime := time.Date(2023, 1, 1, 10, 0, 45, 0, time.UTC) // Still belongs to 10:01 candle
	lateTrade := models.Trade{TradeID: "3", Timestamp: lateTradeTime.Unix(), Price: 90.0, Size: 0.5}
	builder.processTrade(lateTrade)

	// Check the updated 10:01 closed candle
	expectedClosedCandleTime := roundUpTime(tradeTime1, BuilderInterval1m)
	closedCandle, exists := builder.closed[expectedClosedCandleTime]
	require.True(t, exists, "Expected closed candle with timestamp %v not found after late trade", expectedClosedCandleTime)
	require.Equal(t, trade1.Price, closedCandle.Open, "Closed candle Open incorrect")
	require.Equal(t, trade1.Price, closedCandle.High, "Closed candle High incorrect") // High should still be 100, late trade is 90
	require.Equal(t, lateTrade.Price, closedCandle.Low, "Closed candle Low incorrect")
	require.Equal(t, lateTrade.Price, closedCandle.Close, "Closed candle Close incorrect")
	expectedVolume := (trade1.Size * trade1.Price) + (lateTrade.Size * lateTrade.Price)
	require.Equal(t, expectedVolume, closedCandle.Volume, "Closed candle Volume incorrect")

	// Check the current (10:02) candle remains unchanged
	expectedCurrentCandleTime := roundUpTime(tradeTime2, BuilderInterval1m)
	require.Equal(t, expectedCurrentCandleTime, builder.current.Timestamp, "Current candle timestamp changed")
	require.Equal(t, trade2.Price, builder.current.Open, "Current candle Open changed unexpectedly")
	require.Equal(t, trade2.Price, builder.current.High, "Current candle High changed unexpectedly")
	require.Equal(t, trade2.Price, builder.current.Low, "Current candle Low changed unexpectedly")
	require.Equal(t, trade2.Price, builder.current.Close, "Current candle Close changed unexpectedly")
}

func TestProcessTrades_MultipleTrades(t *testing.T) {
	params := CandleBuilderParams{Interval: BuilderInterval1m}
	builder := NewBuilder(params)

	trades := []models.Trade{
		{TradeID: "1", Timestamp: time.Date(2023, 1, 1, 10, 0, 10, 0, time.UTC).Unix(), Price: 100.0, Size: 1.0},
		{TradeID: "2", Timestamp: time.Date(2023, 1, 1, 10, 0, 20, 0, time.UTC).Unix(), Price: 105.0, Size: 2.0},
		{TradeID: "3", Timestamp: time.Date(2023, 1, 1, 10, 1, 5, 0, time.UTC).Unix(), Price: 110.0, Size: 1.5},
		{TradeID: "4", Timestamp: time.Date(2023, 1, 1, 10, 1, 15, 0, time.UTC).Unix(), Price: 108.0, Size: 0.5},
	}

	builder.ProcessTrades(trades)

	// Expected closed candle (10:01)
	expectedClosedCandleTime := roundUpTime(time.Date(2023, 1, 1, 10, 0, 10, 0, time.UTC), BuilderInterval1m)
	closedCandle, exists := builder.closed[expectedClosedCandleTime]
	require.True(t, exists, "Expected closed candle with timestamp %v not found", expectedClosedCandleTime)
	require.Equal(t, 100.0, closedCandle.Open, "Closed candle Open incorrect")
	require.Equal(t, 105.0, closedCandle.High, "Closed candle High incorrect")
	require.Equal(t, 100.0, closedCandle.Low, "Closed candle Low incorrect")
	require.Equal(t, 105.0, closedCandle.Close, "Closed candle Close incorrect")
	require.Equal(t, (100.0*1.0 + 105.0*2.0), closedCandle.Volume, "Closed candle Volume incorrect")

	// Expected current candle (10:02)
	expectedCurrentCandleTime := roundUpTime(time.Date(2023, 1, 1, 10, 1, 5, 0, time.UTC), BuilderInterval1m)
	require.NotNil(t, builder.current, "Expected current candle to be initialized")
	require.Equal(t, expectedCurrentCandleTime, builder.current.Timestamp, "Current candle timestamp mismatch")
	require.Equal(t, 110.0, builder.current.Open, "Current candle Open incorrect")
	require.Equal(t, 110.0, builder.current.High, "Current candle High incorrect")
	require.Equal(t, 108.0, builder.current.Low, "Current candle Low incorrect")
	require.Equal(t, 108.0, builder.current.Close, "Current candle Close incorrect")
	require.Equal(t, (110.0*1.5 + 108.0*0.5), builder.current.Volume, "Current candle Volume incorrect")
}

func TestInitializeCandle(t *testing.T) {
	tradeTime := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	trade := models.Trade{Timestamp: tradeTime.Unix(), Price: 100.0, Size: 1.0}
	candleTime := roundUpTime(tradeTime, BuilderInterval1m)

	candle := initializeCandle(candleTime, trade)

	require.NotNil(t, candle, "initializeCandle returned nil")
	require.Equal(t, candleTime, candle.Timestamp, "Expected timestamp mismatch")
	require.Equal(t, trade.Price, candle.Open, "Open price mismatch")
	require.Equal(t, trade.Price, candle.High, "High price mismatch")
	require.Equal(t, trade.Price, candle.Low, "Low price mismatch")
	require.Equal(t, trade.Price, candle.Close, "Close price mismatch")
	require.Equal(t, trade.Size*trade.Price, candle.Volume, "Volume mismatch")
}

func TestUpdateCandle(t *testing.T) {
	// Test with an empty candle
	candle := &models.Candle{}
	trade1 := models.Trade{Price: 100.0, Size: 1.0}
	updateCandle(candle, trade1)
	expectedVolume1 := trade1.Price * trade1.Size
	require.Equal(t, 100.0, candle.Open, "Failed empty candle update: Open")
	require.Equal(t, 100.0, candle.High, "Failed empty candle update: High")
	require.Equal(t, 100.0, candle.Low, "Failed empty candle update: Low")
	require.Equal(t, 100.0, candle.Close, "Failed empty candle update: Close")
	require.Equal(t, expectedVolume1, candle.Volume, "Failed empty candle update: Volume")

	// Test with a trade increasing high
	trade2 := models.Trade{Price: 120.0, Size: 2.0}
	updateCandle(candle, trade2)
	expectedVolume2 := expectedVolume1 + (trade2.Price * trade2.Size)
	require.Equal(t, 100.0, candle.Open, "Failed high update: Open")
	require.Equal(t, 120.0, candle.High, "Failed high update: High")
	require.Equal(t, 100.0, candle.Low, "Failed high update: Low")
	require.Equal(t, 120.0, candle.Close, "Failed high update: Close")
	require.Equal(t, expectedVolume2, candle.Volume, "Failed high update: Volume")

	// Test with a trade decreasing low
	trade3 := models.Trade{Price: 80.0, Size: 0.5}
	updateCandle(candle, trade3)
	expectedVolume3 := expectedVolume2 + (trade3.Price * trade3.Size)
	require.Equal(t, 100.0, candle.Open, "Failed low update: Open")
	require.Equal(t, 120.0, candle.High, "Failed low update: High")
	require.Equal(t, 80.0, candle.Low, "Failed low update: Low")
	require.Equal(t, 80.0, candle.Close, "Failed low update: Close")
	require.Equal(t, expectedVolume3, candle.Volume, "Failed low update: Volume")

	// Test with a trade within range
	trade4 := models.Trade{Price: 105.0, Size: 1.0}
	updateCandle(candle, trade4)
	expectedVolume4 := expectedVolume3 + (trade4.Price * trade4.Size)
	require.Equal(t, 100.0, candle.Open, "Failed within range update: Open")
	require.Equal(t, 120.0, candle.High, "Failed within range update: High")
	require.Equal(t, 80.0, candle.Low, "Failed within range update: Low")
	require.Equal(t, 105.0, candle.Close, "Failed within range update: Close")
	require.Equal(t, expectedVolume4, candle.Volume, "Failed within range update: Volume")
}

func TestRoundUpTime(t *testing.T) {
	testCases := []struct {
		name     string
		input    time.Time
		interval time.Duration
		expected int64
	}{
		{
			name:     "Round up to next 1-minute",
			input:    time.Date(2023, 1, 1, 10, 0, 59, 999, time.UTC),
			interval: BuilderInterval1m,
			expected: time.Date(2023, 1, 1, 10, 1, 0, 0, time.UTC).Unix(),
		},
		{
			name:     "Round up to next 1-minute, already aligned",
			input:    time.Date(2023, 1, 1, 10, 0, 30, 0, time.UTC),
			interval: BuilderInterval1m,
			expected: time.Date(2023, 1, 1, 10, 1, 0, 0, time.UTC).Unix(),
		},
		{
			name:     "Round up to next 5-minute",
			input:    time.Date(2023, 1, 1, 10, 2, 30, 0, time.UTC),
			interval: BuilderInterval5m,
			expected: time.Date(2023, 1, 1, 10, 5, 0, 0, time.UTC).Unix(),
		},
		{
			name:     "Zero duration",
			input:    time.Date(2023, 1, 1, 10, 0, 30, 0, time.UTC),
			interval: 0,
			expected: time.Date(2023, 1, 1, 10, 0, 30, 0, time.UTC).Unix(),
		},
		{
			name:     "Round up to next 1-hour",
			input:    time.Date(2023, 1, 1, 10, 30, 0, 0, time.UTC),
			interval: BuilderInterval1h,
			expected: time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC).Unix(),
		},
	}

	for _, tc := range testCases {
		got := roundUpTime(tc.input, tc.interval)
		require.Equalf(t, tc.expected, got, "%s", tc.name)
	}
}

func TestBuilder_InitialClosedMap(t *testing.T) {
	params := CandleBuilderParams{Interval: BuilderInterval1m}
	builder := NewBuilder(params)

	require.NotNil(t, builder.closed, "Expected 'closed' map to be initialized")
	require.Empty(t, builder.closed, "Expected 'closed' map to be empty on initialization")
}

func TestProcessTrades(t *testing.T) {
	params := CandleBuilderParams{Interval: BuilderInterval1m}
	builder := NewBuilder(params)

	trades := []models.Trade{
		{TradeID: "1", Timestamp: time.Date(2023, 1, 1, 10, 0, 15, 0, time.UTC).Unix(), Price: 100, Size: 1},
		{TradeID: "2", Timestamp: time.Date(2023, 1, 1, 10, 0, 45, 0, time.UTC).Unix(), Price: 105, Size: 2},
		{TradeID: "3", Timestamp: time.Date(2023, 1, 1, 10, 1, 10, 0, time.UTC).Unix(), Price: 110, Size: 1},
	}

	builder.ProcessTrades(trades)

	// Check the first closed candle (10:01)
	expectedClosedTimestamp1 := time.Date(2023, 1, 1, 10, 1, 0, 0, time.UTC).Unix()
	_, ok := builder.closed[expectedClosedTimestamp1]
	require.True(t, ok, "Expected closed candle for %v", time.Unix(expectedClosedTimestamp1, 0))
	closedCandle := builder.closed[expectedClosedTimestamp1]
	require.Equal(t, float64(100), closedCandle.Open, "Closed candle (10:01) Open incorrect")
	require.Equal(t, float64(105), closedCandle.High, "Closed candle (10:01) High incorrect")
	require.Equal(t, float64(100), closedCandle.Low, "Closed candle (10:01) Low incorrect")
	require.Equal(t, float64(105), closedCandle.Close, "Closed candle (10:01) Close incorrect")
	require.Equal(t, float64(100*1+105*2), closedCandle.Volume, "Closed candle (10:01) Volume incorrect")

	// Check the current candle (10:02)
	expectedCurrentTimestamp := time.Date(2023, 1, 1, 10, 2, 0, 0, time.UTC).Unix()
	require.NotNil(t, builder.current, "Expected current candle to be initialized")
	require.Equal(t, expectedCurrentTimestamp, builder.current.Timestamp, "Current candle (10:02) Timestamp incorrect")
	require.Equal(t, float64(110), builder.current.Open, "Current candle (10:02) Open incorrect")
	require.Equal(t, float64(110), builder.current.High, "Current candle (10:02) High incorrect")
	require.Equal(t, float64(110), builder.current.Low, "Current candle (10:02) Low incorrect")
	require.Equal(t, float64(110), builder.current.Close, "Current candle (10:02) Close incorrect")
	require.Equal(t, float64(110*1), builder.current.Volume, "Current candle (10:02) Volume incorrect")
}
