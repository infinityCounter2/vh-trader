package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/infinityCounter2/vh-trader/internal/logic"
	"github.com/infinityCounter2/vh-trader/internal/models"
	"github.com/mailru/easyjson"
)

type Params struct {
	Port int
}

type Server struct {
	p Params

	// The server handles deduping of trades
	// from input itself but in a production system
	// this should all be abstracted away.
	knwnMtx       sync.Mutex
	knownTradeIDs map[string]struct{}
	// tradeStore will store all the trades ingested
	// and also be called to respond to requests to get
	// trades for a symbol
	tradeStore *logic.TradeStore

	builderMtx sync.RWMutex
	// builders is keyed "symbol_interval" and contains
	// all the candle builders
	builders map[string]*logic.CandleBuilder
}

func NewServer(p Params) *Server {
	// Standard HTTP Mux server, no need for anything fancy
	return &Server{
		p:             p,
		knwnMtx:       sync.Mutex{},
		knownTradeIDs: make(map[string]struct{}),
		tradeStore: logic.NewTradeStore(logic.TradeStoreParams{
			CacheLimit: 50,
		}),
		builders:   make(map[string]*logic.CandleBuilder),
		builderMtx: sync.RWMutex{},
	}
}

// Run starts the HTTP server and will continue until either an
// expected event is encountered, or the provided context is finished.
func (s *Server) Run(ctx context.Context) error {
	// Buffer the error channel so that the routine
	// pushing to it can exit immediately.
	errCh := make(chan error, 1)

	mux := http.NewServeMux()

	mux.HandleFunc("/ingest", s.ingestHandler)
	mux.HandleFunc("/trades", s.tradesHandler)
	mux.HandleFunc("/candles", s.candlesHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.p.Port),
		Handler: middleware(mux),
	}

	// Start serving.
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err // Abnormal termination event so push the error
			return
		}
		errCh <- nil // http.ErrServerClosed is a normal shutdown event
	}()

	// Wait for the context to end
	select {
	case <-ctx.Done():
		// Attempt a graceful shutdown with a timeout
		shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = srv.Shutdown(shCtx) // We wll drop the error here since it's inconsequential
		<-errCh
		return nil

	case err := <-errCh:
		// Non-graceful server error (bind failure, etc.)
		return err
	}
}

// ingestHandler is a handler for the /ingest endpoint to ingest trades for processing
//
// Only handles POST requests
func (s *Server) ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load and parse JSON body
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read POST body", http.StatusInternalServerError)
		return
	}

	defer r.Body.Close()

	var trades models.TradeList
	if err := easyjson.Unmarshal(payload, &trades); err != nil {
		http.Error(w, "Failed to parsed POST body to trades", http.StatusUnprocessableEntity)
		return
	}

	if len(trades) == 0 {
		w.Write([]byte("Processed 0 trades!"))
		return
	}

	dedupedTrades := make([]models.Trade, 0, len(trades))
	tradesBySymbol := make(map[string][]models.Trade)

	s.knwnMtx.Lock()
	for _, t := range trades {
		if _, seen := s.knownTradeIDs[t.TradeID]; seen {
			// Dedup trades
			continue
		}
		s.knownTradeIDs[t.TradeID] = struct{}{}
		dedupedTrades = append(dedupedTrades, t)
		tradesBySymbol[t.Symbol] = append(tradesBySymbol[t.Symbol], t)
	}
	s.knwnMtx.Unlock()

	s.tradeStore.PushTrades(dedupedTrades)

	// On a symbol by symbol basis process the batch of trades
	for symbol, trades := range tradesBySymbol {
		for _, intvl := range logic.BuilderIntervals {
			// For each interval, pass the trades to the builder
			builderKey := getBuilderKey(symbol, intvl)

			s.builderMtx.Lock()
			builder, ok := s.builders[builderKey]
			if !ok {
				// If no builder exists for the symbol, initialize one
				builder = logic.NewBuilder(logic.CandleBuilderParams{
					Interval: intvl,
				})
				s.builders[builderKey] = builder
			}
			s.builderMtx.Unlock()

			builder.ProcessTrades(trades)
		}
	}

	w.Write([]byte(fmt.Sprintf("Processs %d of %d trades!", len(dedupedTrades), len(trades))))
}

// tradesHandler is a handler for the /trades endpoint to server the 50 latest
// trades on a GET request for any given "symbol".
func (s *Server) tradesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	symbol := getParam(r, "symbol")
	if symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}

	trades := s.tradeStore.GetTrades(symbol)
	if trades == nil {
		trades = make([]models.Trade, 0)
	}

	writeJSON(w, models.TradeList(trades))
}

// candlesHandler is a handler for the /candle endpoint to server aggregated
// OHLC candle based on the required "symbol" and optional "interval" (defaults to 1m)
// parameters.
func (s *Server) candlesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	symbol := getParam(r, "symbol")
	if symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}

	intvlArg := getParamOr(r, "interval", "1m")
	intvl, ok := parseBuilderInterval(intvlArg)
	if !ok {
		http.Error(w,
			fmt.Sprintf("invalid interval value %q", intvlArg),
			http.StatusBadRequest,
		)
		return
	}

	builderKey := getBuilderKey(symbol, intvl)
	s.builderMtx.RLock()
	builder := s.builders[builderKey]
	s.builderMtx.RUnlock()

	var candles models.CandleList
	if builder != nil {
		// There are no candles in this interval for this symbol
		candles = builder.GetCandles()
	} else {
		candles = make(models.CandleList, 0)
	}

	writeJSON(w, candles)
}

// writeJSON is a helper for serializing the response via easyjson
// and writing it back to the client.
func writeJSON(w http.ResponseWriter, data easyjson.Marshaler) {
	payload, err := easyjson.Marshal(data)
	if err != nil {
		fmt.Printf("Failed to marshal response: %s\n", err)
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		fmt.Printf("Failed to write response to client: %s\n", err)
	}
}

// getParam retrieves a query parameter from the request URL.
// It returns the parameter's value as a string. If the parameter is not found,
// an empty string is returned.
func getParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// getParamOr retrieves a query parameter from the request URL.
// It returns the parameter's value as a string. If the parameter is not found,
// the provided defaultValue is returned instead.
func getParamOr(r *http.Request, key, defVal string) string {
	val := getParam(r, key)
	if val == "" {
		return defVal
	}
	return val
}

// Simple request logging and validation middleware.
//
// This would be available as built in by some frameworks
// such as GIN.
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
	})
}

var validIntervals = map[string]logic.BuilderInterval{
	"1m":  logic.BuilderInterval1m,
	"5m":  logic.BuilderInterval5m,
	"15m": logic.BuilderInterval15m,
	"1h":  logic.BuilderInterval1h,
}

func parseBuilderInterval(k string) (logic.BuilderInterval, bool) {
	intvl, ok := validIntervals[k]
	return intvl, ok
}

func getBuilderKey(symbol string, intvl logic.BuilderInterval) string {
	return fmt.Sprintf("%s_%s", symbol, intvl)
}
