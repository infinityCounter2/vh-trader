# Homma Project

## Overview

This project, named `homma`, is a simple OHLC candle builder for trades. It features a REST API with endpoints `/trades` and `/ohlc` for viewing data.

## Getting Started

### Prerequisites

Make sure you have Go installed on your system. You can download it from [golang.org](https://golang.org/doc/install).

This project uses `easyjson` for JSON serialization. If you don't have it installed, the `generate` command in the Makefile will attempt to install it for you.

### Building the Project

To compile the `homma` binary, run the following command:

```bash
make build
```

The compiled binary will be located in the `bin/` directory.

### Generating Code

To run `go generate` across all subfolders and generate necessary code (e.g., `easyjson` serialization files), use:

```bash
make generate
```

This command will also check for and install `easyjson` if it's not already present.

### Running the Application

To first generate code, then build the binary, and finally execute the `homma` application, run:

```bash
make run
```

### Cleaning Up

To remove the compiled binary and other generated files, use:

```bash
make clean
```
