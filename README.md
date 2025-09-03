# Homma Project

## Overview

This project, named `homma`, is a simple OHLC candle builder for trades. It features a REST API with endpoints `/ingest`, `/trades` and `/candles` for viewing and ingesting data.

## Getting Started

### Prerequisites

Make sure you have Go installed on your system. You can download it from [golang.org](https://golang.org/doc/install).

This project uses `easyjson` for JSON serialization. If you don't have it installed, the `generate` command in the Makefile will attempt to install it for you.

## API Endpoints

### `POST /ingest`

Used to ingest trade data into the system. The request body should be a JSON array of trade objects. An usable test data set exists in [trades.json](/internal/testdata/trades.json)

Example:
```json
[
  {
    "trade_id": "123",
    "symbol": "BTC_USD",
    "timestamp": 1672531200,
    "price": 16500.50,
    "size": 0.1
  },
  {
    "trade_id": "124",
    "symbol": "BTC_USD",
    "timestamp": 1672531210,
    "price": 16501.00,
    "size": 0.05
  }
]
```

### `GET /trades`

Retrieves the most recent trades for a given symbol. The trades are returned in oldest-to-newest order, up to a maximum of 50 trades.

**Query Parameters:**
- `symbol` (required): The trading pair symbol (e.g., `BTC_USD`).

Example:
```
GET /trades?symbol=BTC_USD
```

### `GET /candles`

Retrieves OHLC (Open, High, Low, Close) candles for a given symbol and interval. Candles are returned in oldest-to-newest order.

**Query Parameters:**
- `symbol` (required): The trading pair symbol (e.g., `BTC_USD`).
- `interval` (optional): The candle interval. Supported values: `1m`, `5m`, `15m`, `1h`. Defaults to `1m`.

Example:
```
GET /candles?symbol=BTC_USD&interval=5m
```

## Building the Project

To compile the `homma` binary, run the following command:

```bash
make build
```

The compiled binary will be located in the `bin/` directory.

## Generating Code

To run `go generate` across all subfolders and generate necessary code (e.g., `easyjson` serialization files), use:

```bash
make generate
```

This command will also check for and install `easyjson` if it's not already present.

## Running the Application

To first generate code, then build the binary, and finally execute the `homma` application, run:

```bash
make run
```

## Cleaning Up

To remove the compiled binary and other generated files, use:

```bash
make clean
```
