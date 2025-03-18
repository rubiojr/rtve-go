# RTVE Scraper

A command-line tool for downloading subtitles and video metadata from RTVE (Radio Televisión Española).

## Features

- Scrape videos from RTVE show pages
- Download video metadata in JSON format
- Download subtitles in VTT format
- Organize videos by publication date
- Support for rate limiting and pagination
- Command-line interface with customizable options

## Installation

### Prerequisites

- Go 1.23 or higher

### Building from source

```
go install github.com/rubiojr/rtve-go/cmd/rtve-subs@latest
```

## Usage

```bash
rtve-subs fetch --show telediario-1

# Specify output directory
rtve-subs --output="/path/to/videos" --show telediario-1

# Enable verbose output
rtve-subs --verbose --show telediario-1
```

## Supported Shows

Currently supported shows include:
- telediario-1
- telediario-2
- informe-semanal

Use the `list-shows` command to see the complete list of available shows

### Command-line Options

| Option | Alias | Default | Description |
|--------|-------|---------|-------------|
| `--output` | `-o` | `rtve-videos` | Output directory for downloaded content |
| `--show` | `-p` | none | Show to scrap (required) |
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

1. The scraper navigates through pages of the specified RTVE show URL
2. For each page, it extracts links to individual videos
3. For each video:
   - Metadata is downloaded and saved as JSON
   - Subtitles are downloaded if available
   - Content is organized by publication date

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This tool is provided for personal use only. Be mindful of RTVE's terms of service and copyright restrictions when using this tool. The developers are not responsible for any misuse of this software.
