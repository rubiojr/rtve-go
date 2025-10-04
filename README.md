# RTVE Scraper

[![Go Reference](https://pkg.go.dev/badge/github.com/rubiojr/rtve-go.svg)](https://pkg.go.dev/github.com/rubiojr/rtve-go)

A Go library and command-line tool for downloading subtitles and video metadata from RTVE (Radio Televisión Española).

## Features

- **Go API**: Programmatic access to RTVE video metadata and subtitles
- **CLI Tool**: Command-line interface for downloading content
- Scrape videos from RTVE show pages
- Download video metadata in JSON format
- Download subtitles in VTT format (multiple languages)
- Organize videos by publication date
- Support for pagination and date range filtering
- Fetch latest videos from one or all shows

## Installation

### Prerequisites

- Go 1.23 or higher

### Building from source

```bash
go install github.com/rubiojr/rtve-go/cmd/rtve-subs@latest
```

Or clone and build:

```bash
git clone https://github.com/rubiojr/rtve-go
cd rtve-go
go build -o rtve-subs cmd/rtve-subs/main.go
```

## Usage

### Command Line Interface

#### Fetch videos from a specific show

```bash
rtve-subs fetch --show telediario-1

# Specify output directory
rtve-subs fetch --output="/path/to/videos" --show telediario-1

# Fetch multiple pages
rtve-subs fetch --show telediario-1 --max-pages 5

# Enable verbose output
rtve-subs fetch --show telediario-1 --verbose
```

#### Fetch latest videos

```bash
# Fetch the latest video from a specific show
rtve-subs fetch-latest --show telediario-1 --count 1

# Fetch latest 5 videos from a show
rtve-subs fetch-latest --show telediario-1 --count 5

# Fetch latest video from ALL available shows
rtve-subs fetch-latest --count 1

# Fetch latest 3 videos from all shows
rtve-subs fetch-latest --count 3
```

#### List available shows

```bash
rtve-subs list-shows
```

### Go API

The package also provides a programmatic API for Go applications:

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/rubiojr/rtve-go/api"
)

func main() {
    // Define date range
    start := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)
    end := time.Date(2025, 10, 5, 23, 59, 59, 0, time.UTC)
    
    // Create visitor function to process each video
    visitor := func(result *api.VideoResult) error {
        fmt.Printf("Title: %s\n", result.Metadata.LongTitle)
        fmt.Printf("Published: %s\n", result.Metadata.PublicationDate)
        
        if result.Subtitles != nil {
            fmt.Printf("Subtitles: %d tracks\n", len(result.Subtitles.Subtitles))
        }
        return nil
    }
    
    // Fetch videos
    stats, err := api.FetchShow("telediario-1", start, end, visitor)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Processed %d videos\n", stats.VideosProcessed)
}
```

See the [API documentation](https://pkg.go.dev/github.com/rubiojr/rtve-go/api) for more details.

## Supported Shows

Currently supported shows include:
- `telediario-1` - Telediario 15 horas
- `telediario-2` - Telediario 21 horas
- `telediario-matinal` - Telediario Matinal
- `informe-semanal` - Informe Semanal

Use the `list-shows` command to see the complete list of available shows.

### Command-line Options

#### `fetch` command

| Option | Alias | Default | Description |
|--------|-------|---------|-------------|
| `--output` | `-o` | `rtve-videos` | Output directory for downloaded content |
| `--show` | `-s` | (required) | Show to scrape |
| `--max-pages` | `-m` | `1` | Maximum number of pages to scrape |
| `--verbose` | `-v` | `false` | Enable verbose output |

#### `fetch-latest` command

| Option | Alias | Default | Description |
|--------|-------|---------|-------------|
| `--output` | `-o` | `rtve-videos` | Output directory for downloaded content |
| `--show` | `-s` | (optional) | Show to fetch (if not specified, fetches from all shows) |
| `--count` | `-n` | `1` | Number of latest videos to fetch per show |
| `--verbose` | `-v` | `false` | Enable verbose output |

## Output Structure

The scraper organizes videos by year and date:

```
rtve-videos/
  ├── 2023/
  │   ├── 2023-01-01/
  │   │   ├── video_12345.json
  │   │   └── subs/
  │   │       ├── 12345_es.vtt
  │   │       └── 12345_en.vtt
  │   └── 2023-01-02/
  │       └── ...
  └── 2022/
      └── ...
```

## How It Works

### Scraper (fetch command)
1. Navigates through pages of the specified RTVE show URL
2. Extracts video links using regex patterns
3. For each video:
   - Downloads metadata from RTVE's JSON API
   - Downloads available subtitles (Spanish, English, Catalan, Basque, Galician)
   - Organizes content by publication date
   - Sets folder timestamps to match publication date

### API (fetch-latest, FetchShow)
1. Uses the high-level API package for cleaner integration
2. Supports date range filtering
3. Processes videos through a visitor function pattern
4. Provides detailed statistics about the fetch operation

## Development

### Running Tests

```bash
# Run unit tests (fast)
go test ./...

# Run integration tests (requires network access)
go test ./api -v -run TestIntegration
```

### CI/CD

The project uses GitHub Actions for continuous integration:

- **On Push/PR**: Runs unit tests and builds the binary
- **Weekly (Monday 9 AM UTC)**: Runs integration tests against RTVE's live API
- **On Integration Test Failure**: Automatically creates a GitHub issue

The weekly integration tests help detect when RTVE changes their API or website structure, allowing for proactive maintenance.

## API Documentation

Full API documentation is available at [pkg.go.dev/github.com/rubiojr/rtve-go/api](https://pkg.go.dev/github.com/rubiojr/rtve-go/api).

Key types and functions:
- `FetchShow(showID, startDate, endDate, visitor)` - Fetch videos within a date range
- `FetchShowLatest(showID, maxVideos, visitor)` - Fetch the most recent videos
- `FetchShowAll(showID, visitor)` - Fetch all available videos
- `AvailableShows()` - Get list of supported shows
- `VideoResult` - Contains metadata and subtitles for a video
- `VisitorFunc` - Function type for processing each video

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This tool is provided for personal use only. Be mindful of RTVE's terms of service and copyright restrictions when using this tool. The developers are not responsible for any misuse of this software.
