package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"slices"

	"github.com/rubiojr/rtve-go"
	"github.com/rubiojr/rtve-go/api"
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
						Value:   0,
						Usage:   "Maximum number of pages to scrape (0 = unlimited)",
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
				Name:   "fetch-latest",
				Usage:  "Fetch the latest available video(s) from RTVE",
				Action: fetchLatest,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Value:   "rtve-videos",
						Usage:   "Output directory for downloaded content",
					},
					&cli.StringFlag{
						Name:    "show",
						Aliases: []string{"s"},
						Usage:   "Show to fetch (if not specified, fetches latest from all shows)",
					},
					&cli.IntFlag{
						Name:    "count",
						Aliases: []string{"n"},
						Value:   1,
						Usage:   "Number of latest videos to fetch per show",
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
	if maxPages == 0 {
		fmt.Printf("Max pages: unlimited\n")
	} else {
		fmt.Printf("Max pages: %d\n", maxPages)
	}

	shows := rtve.ListShows()
	if !slices.Contains(shows, show) {
		return fmt.Errorf("unsupported show: %s", show)
	}

	// Create the scraper with the provided options
	scrapper := rtve.NewScrapper(
		show,
		rtve.WithOutputPath(outputPath),
		rtve.WithVerbose(verbose),
	)

	// Start scraping
	startTime := time.Now()
	videosDownloaded, errs := scrapper.Scrape(maxPages)

	if verbose {
		for _, err := range errs {
			fmt.Printf("Error: %v\n", err)
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("\nScraping completed in %s\n", duration)
	fmt.Printf("Downloaded %d videos\n", videosDownloaded)

	return nil
}

func fetchLatest(c *cli.Context) error {
	outputPath := c.String("output")
	show := c.String("show")
	count := c.Int("count")
	verbose := c.Bool("verbose")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	fmt.Printf("Fetching latest videos from RTVE\n")
	fmt.Printf("Output directory: %s\n", outputPath)

	var showsToFetch []string
	if show != "" {
		// Validate show
		availableShows := api.AvailableShows()
		if !slices.Contains(availableShows, show) {
			return fmt.Errorf("unsupported show: %s (use list-shows to see available shows)", show)
		}
		showsToFetch = []string{show}
		fmt.Printf("Show: %s\n", show)
	} else {
		// Fetch from all shows
		showsToFetch = api.AvailableShows()
		fmt.Printf("Fetching from all shows\n")
	}
	fmt.Printf("Count per show: %d\n\n", count)

	totalVideos := 0
	totalErrors := 0

	for _, showID := range showsToFetch {
		if len(showsToFetch) > 1 {
			fmt.Printf("\n--- Fetching from %s ---\n", showID)
		}

		showVideos := 0

		visitor := func(result *api.VideoResult) error {
			showVideos++

			// Create folder structure based on publication date
			folder, err := createFolderForVideo(result.Metadata, outputPath)
			if err != nil {
				if verbose {
					fmt.Printf("Error creating folder for %s: %v\n", result.Metadata.ID, err)
				}
				return nil // Continue processing
			}

			// Save video metadata
			if err := saveVideoMetadata(result.Metadata, folder); err != nil {
				if verbose {
					fmt.Printf("Error saving metadata for %s: %v\n", result.Metadata.ID, err)
				}
				totalErrors++
				return nil // Continue processing
			}

			// Save subtitles if available
			if result.Subtitles != nil {
				if err := saveSubtitles(result.Subtitles, folder); err != nil {
					if verbose {
						fmt.Printf("Error saving subtitles for %s: %v\n", result.Metadata.ID, err)
					}
					totalErrors++
				}
			}

			// Set folder modification time
			if err := updateFolderTime(result.Metadata, folder); err != nil {
				if verbose {
					fmt.Printf("Error updating folder time for %s: %v\n", result.Metadata.ID, err)
				}
			}

			fmt.Printf("✓ Downloaded: %s (ID: %s)\n", result.Metadata.LongTitle, result.Metadata.ID)
			if result.Subtitles != nil {
				fmt.Printf("  Subtitles: %d track(s)\n", len(result.Subtitles.Subtitles))
			}

			return nil
		}

		stats, err := api.FetchShowLatest(showID, count, visitor)
		if err != nil {
			fmt.Printf("Error fetching %s: %v\n", showID, err)
			totalErrors++
			continue
		}

		totalVideos += stats.VideosProcessed
		if len(stats.Errors) > 0 && verbose {
			fmt.Printf("Non-fatal errors for %s:\n", showID)
			for _, e := range stats.Errors {
				fmt.Printf("  - %v\n", e)
			}
		}

		if showVideos == 0 {
			fmt.Printf("No videos found for %s\n", showID)
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total videos downloaded: %d\n", totalVideos)
	fmt.Printf("Total errors: %d\n", totalErrors)

	return nil
}

func createFolderForVideo(meta *rtve.VideoMetadata, basePath string) (string, error) {
	layout := "02-01-2006 15:04:05"
	pubDate, err := time.Parse(layout, meta.PublicationDate)
	if err != nil {
		return "", fmt.Errorf("parsing publication date: %w", err)
	}

	folder := filepath.Join(basePath, pubDate.Format("2006"), pubDate.Format("2006-01-02"))
	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", fmt.Errorf("creating folder: %w", err)
	}

	return folder, nil
}

func saveVideoMetadata(meta *rtve.VideoMetadata, folder string) error {
	jsonData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	filename := filepath.Join(folder, fmt.Sprintf("video_%s.json", meta.ID))
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("writing metadata file: %w", err)
	}

	return nil
}

func saveSubtitles(subs *rtve.Subtitles, folder string) error {
	subsDir := filepath.Join(folder, "subs")
	if err := os.MkdirAll(subsDir, 0755); err != nil {
		return fmt.Errorf("creating subs directory: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	for _, sub := range subs.Subtitles {
		// Download subtitle content
		resp, err := client.Get(sub.Src)
		if err != nil {
			return fmt.Errorf("downloading subtitle %s: %w", sub.Lang, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("subtitle download for %s returned status %d", sub.Lang, resp.StatusCode)
		}

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading subtitle %s: %w", sub.Lang, err)
		}

		filename := filepath.Join(subsDir, fmt.Sprintf("%s_%s.vtt", subs.VideoID, sub.Lang))
		if err := os.WriteFile(filename, content, 0644); err != nil {
			return fmt.Errorf("writing subtitle file: %w", err)
		}
	}

	return nil
}

func updateFolderTime(meta *rtve.VideoMetadata, folder string) error {
	layout := "02-01-2006 15:04:05"
	pubDate, err := time.Parse(layout, meta.PublicationDate)
	if err != nil {
		return fmt.Errorf("parsing publication date: %w", err)
	}

	return os.Chtimes(folder, pubDate, pubDate)
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
