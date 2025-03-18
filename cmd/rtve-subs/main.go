package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/rubiojr/rtve-go"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "rtve-scraper",
		Usage: "Download videos and subtitles from RTVE",
		Commands: []*cli.Command{
			{
				Name:   "fetch",
				Usage:  "Download videos from RTVE",
				Action: runScraper,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Value:   "rtve-videos",
						Usage:   "Output directory for downloaded content",
					},
					&cli.StringFlag{
						Name:     "show",
						Aliases:  []string{"p"},
						Required: true,
						Usage:    "Show to scrape",
					},
					&cli.IntFlag{
						Name:    "max-pages",
						Aliases: []string{"m"},
						Value:   1,
						Usage:   "Maximum number of pages to scrape",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"v"},
						Value:   false,
						Usage:   "Enable verbose output",
					},
				},
			},
			{
				Name:   "list-shows",
				Usage:  "List available shows that can be downloaded",
				Action: listShows,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func runScraper(c *cli.Context) error {
	outputPath := c.String("output")
	show := c.String("show")
	maxPages := c.Int("max-pages")
	verbose := c.Bool("verbose")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	fmt.Printf("Starting RTVE scraper\n")
	fmt.Printf("Output directory: %s\n", outputPath)
	fmt.Printf("Show: %s\n", show)
	fmt.Printf("Max pages: %d\n", maxPages)

	switch show {
	case "telediario-1", "telediario-2", "informe-semanal":
	default:
		return fmt.Errorf("unsupported show: %s", show)
	}

	// Create the scraper with the provided options
	scraper := rtve.NewScrapper(
		show,
		rtve.WithOutputPath(outputPath),
	)

	// Start scraping
	startTime := time.Now()
	videosDownloaded := scraper.RunWithLimit(maxPages, verbose)

	// Print summary
	duration := time.Since(startTime)
	fmt.Printf("\nScraping completed in %s\n", duration)
	fmt.Printf("Downloaded %d videos\n", videosDownloaded)

	return nil
}

func listShows(c *cli.Context) error {
	fmt.Println("Available shows:")

	shows := rtve.ListShows()
	sort.Strings(shows)

	// Print each show with its details
	for _, show := range shows {
		fmt.Printf("- %s (ID: %s)\n", show, rtve.ShowMap(show).ID)
	}

	fmt.Println("\nUse the show name with the fetch command:")
	fmt.Println("Example: rtve-scraper fetch --show telediario-1")

	return nil
}
