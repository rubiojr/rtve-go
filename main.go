package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

type VideoInfo struct {
	URL string
	ID  string
}

type Scrapper struct {
	Program    string
	client     *http.Client
	outputPath string
}

type Option func(*Scrapper)

func WithOutputPath(path string) Option {
	return func(s *Scrapper) {
		s.outputPath = path
	}
}

func NewScrapper(program string, options ...Option) *Scrapper {
	// Create a new HTTP client
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	s := &Scrapper{
		Program:    program,
		client:     client,
		outputPath: "rtve-videos",
	}

	for _, option := range options {
		option(s)
	}

	return s
}

var ErrPageNotFound = errors.New("error parsing page")

func (s *Scrapper) get(url string) (string, error) {
	// Create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	// Execute the request
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error executing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", ErrPageNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	return string(body), nil
}

func (s *Scrapper) scrape(page int) ([]*VideoInfo, error) {
	content, err := s.get(fmt.Sprintf(urlMap[s.Program].URL, page))
	if err != nil {
		return nil, fmt.Errorf("error downloading HTML: %w", err)
	}

	pattern := regexp.MustCompile(urlMap[s.Program].Regex)

	matches := pattern.FindAllString(content, -1)

	uniqueLinks := make(map[string]bool)
	for _, link := range matches {
		if strings.HasSuffix(link, "/") {
			link = link[:len(link)-1]
		}
		uniqueLinks[link] = true
	}

	var result []*VideoInfo
	for link := range uniqueLinks {
		tokens := strings.Split(link, "/")
		id := tokens[len(tokens)-1]

		result = append(result, &VideoInfo{URL: link, ID: id})
	}

	return result, nil
}

func (s *Scrapper) folderForVideo(meta *VideoMetadata) (string, error) {
	layout := "02-01-2006 15:04:05"
	pubDate, err := time.Parse(layout, meta.PublicationDate)
	if err != nil {
		return "", err
	}

	return filepath.Join(s.outputPath, pubDate.Format("2006"), pubDate.Format("2006-01-02")), nil
}

func (s *Scrapper) checkVideoExists(meta *VideoMetadata) bool {
	folder, err := s.folderForVideo(meta)
	if err != nil {
		log.Printf("error creating folder for video: %w", err)
		return false
	}

	if _, err := os.Stat(folder); !os.IsNotExist(err) {
		return true
	}
	return false
}

func (s *Scrapper) updateFolderTime(meta *VideoMetadata, folder string) error {
	if meta.PublicationDate != "" {
		layout := "02-01-2006 15:04:05"
		pubDate, err := time.Parse(layout, meta.PublicationDate)
		if err != nil {
			return fmt.Errorf("parsing publication date for %s: %w", meta.ID, err)
		} else {
			// Set folder modification time
			err = os.Chtimes(folder, pubDate, pubDate)
			if err != nil {
				return fmt.Errorf("setting folder modification time for %s: %w", meta.ID, err)
			}
		}
	}
	return nil
}

func main() {
	app := &cli.App{
		Name:  "rtve-scraper",
		Usage: "Download videos and subtitles from RTVE",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "rtve-videos",
				Usage:   "Output directory for downloaded content",
			},
			&cli.StringFlag{
				Name:    "program",
				Aliases: []string{"p"},
				Value:   "telediario-1",
				Usage:   "Program to scrape",
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
		Action: runScraper,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func runScraper(c *cli.Context) error {
	outputPath := c.String("output")
	program := c.String("program")
	maxPages := c.Int("max-pages")
	verbose := c.Bool("verbose")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	fmt.Printf("Starting RTVE scraper\n")
	fmt.Printf("Output directory: %s\n", outputPath)
	fmt.Printf("Program: %s\n", program)
	fmt.Printf("Max pages: %d\n", maxPages)

	switch program {
	case "telediario-1":
		// Handle program1
	case "telediario-2":
		// Handle program2
	default:
		return fmt.Errorf("unsupported program: %s", program)
	}

	// Create the scraper with the provided options
	scraper := NewScrapper(
		program,
		WithOutputPath(outputPath),
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

func (s *Scrapper) RunWithLimit(maxPages int, verbose bool) int {
	videosDownloaded := 0

	for page := 0; page <= maxPages; page++ {
		if verbose {
			fmt.Printf("Scraping page %d...\n", page)
		}

		links, err := s.scrape(page)
		// We're done paginating, return
		if errors.Is(err, ErrPageNotFound) {
			break
		}

		if err != nil {
			log.Printf("error finding links on page %d: %v", page, err)
			continue
		}

		if verbose {
			fmt.Printf("Found %d links on page %d\n", len(links), page)
		}

		for _, link := range links {
			meta, err := s.DownloadVideoMeta(link.ID)
			if err != nil {
				fmt.Printf("Error downloading video metadata for %s: %v\n", link.ID, err)
				continue
			}

			// Check if video already exists
			if s.checkVideoExists(meta) {
				if verbose {
					fmt.Printf("Video %s already exists, skipping\n", meta.LongTitle)
				}
				continue
			}

			folder, err := s.folderForVideo(meta)
			if err != nil {
				fmt.Printf("Error creating folder for %s: %v\n", link.ID, err)
				continue
			}
			if err := os.MkdirAll(folder, 0755); err != nil {
				fmt.Printf("Error creating folder for %s: %v\n", link.ID, err)
				continue
			}

			err = s.SaveVideoToFile(meta, folder)
			if err != nil {
				fmt.Printf("Error saving video metadata for %s: %v\n", link.ID, err)
				continue
			}

			err = s.DownloadSubtitles(meta, folder)
			if err != nil {
				fmt.Printf("Error downloading subtitles for %s: %v\n", link.ID, err)
			}

			err = s.updateFolderTime(meta, folder)
			if err != nil {
				fmt.Printf("Error updating folder time for %s: %v\n", link.ID, err)
			}

			fmt.Printf("Downloaded video %s\n", meta.LongTitle)
			videosDownloaded++
		}
	}

	return videosDownloaded
}
