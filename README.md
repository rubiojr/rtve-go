# RTVE Video Scraper

A command-line tool for downloading videos and subtitles from RTVE (Radio Televisión Española).

## Features

- Scrape videos from RTVE program pages
- Download video metadata in JSON format
- Download subtitles in VTT format
- Organize videos by publication date
- Support for rate limiting and pagination
- Command-line interface with customizable options

## Installation

### Prerequisites

- Go 1.16 or higher
- Git

### Building from source

```bash
# Clone the repository
git clone https://github.com/yourusername/rtve-go.git
cd rtve-go

# Build the application
go build -o rtve-scraper

# Optional: Move to a directory in your PATH
sudo mv rtve-scraper /usr/local/bin/
```

## Usage

```bash
# Basic usage (uses default program telediario-1)
./rtve-scraper

# Specify output directory
./rtve-scraper --output="/path/to/videos"

# Use a custom program URL (must contain %d for page number)
./rtve-scraper --program="https://www.rtve.es/play/videos/modulos/capitulos/12345/?page=%d"

# Limit the number of pages to scrape
./rtve-scraper --max-pages=10

# Enable verbose output
./rtve-scraper --verbose
```

### Command-line Options

| Option | Alias | Default | Description |
|--------|-------|---------|-------------|
| `--output` | `-o` | `rtve-videos` | Output directory for downloaded content |
| `--program` | `-p` | `telediario-1` | Program to scrap |
| `--max-pages` | `-m` | `1024` | Maximum number of pages to scrape |
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

1. The scraper navigates through pages of the specified RTVE program URL
2. For each page, it extracts links to individual videos
3. For each video:
   - Metadata is downloaded and saved as JSON
   - Subtitles are downloaded if available
   - Content is organized by publication date

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This tool is provided for personal use only. Be mindful of RTVE's terms of service and copyright restrictions when using this tool. The developers are not responsible for any misuse of this software.
