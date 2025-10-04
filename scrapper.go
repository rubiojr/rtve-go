package rtve

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DownloadVideoMeta fetches and parses video metadata for a given video ID
func (s *Scrapper) DownloadVideoMeta(videoID string) (*VideoMetadata, error) {
	url := fmt.Sprintf(ApiURL, videoID)

	body, err := s.get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching video metadata: %v", err)
	}

	m := &VideoMetadata{}

	return m, m.Parse(body)
}

func (s *Scrapper) SaveVideoToFile(meta *VideoMetadata, directory string) error {
	jsonData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal video metadata: %v", err)
	}

	// Create filename based on video ID
	filename := fmt.Sprintf("%s/video_%s.json", directory, meta.ID)

	// Write to file
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write video metadata to file: %v", err)
	}

	return nil
}

func (s *Scrapper) get(url string) (string, error) {
	const maxRetries = 3
	const initialBackoff = 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
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

		// Check status code
		if resp.StatusCode == 404 {
			resp.Body.Close()
			return "", ErrPageNotFound
		}

		if resp.StatusCode == 403 {
			resp.Body.Close()
			return "", ErrForbidden
		}

		// Retry on 5xx errors
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			resp.Body.Close()
			if attempt < maxRetries {
				backoff := initialBackoff * time.Duration(1<<uint(attempt))
				if s.verbose {
					fmt.Printf("Server error %d, retrying in %v (attempt %d/%d)...\n", resp.StatusCode, backoff, attempt+1, maxRetries)
				}
				time.Sleep(backoff)
				continue
			}
			return "", fmt.Errorf("server error after %d retries: status code %d", maxRetries, resp.StatusCode)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("error reading response body: %w", err)
		}

		return string(body), nil
	}

	return "", fmt.Errorf("unexpected error in retry loop")
}

func (s *Scrapper) ScrapePage(page int) ([]*VideoInfo, error) {
	content, err := s.get(fmt.Sprintf(urlMap[s.Program].URL, page))
	if err != nil {
		return nil, fmt.Errorf("error downloading HTML: %w", err)
	}
	return s.scrape(content)
}

func (s *Scrapper) scrape(content string) ([]*VideoInfo, error) {
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
		return false
	}

	if _, err := os.Stat(folder); !os.IsNotExist(err) {
		return true
	}
	return false
}

// checkVideoExistsByID checks if a video exists by searching for its JSON file
// and returns the folder path if found. This is more efficient than fetching metadata first.
func (s *Scrapper) checkVideoExistsByID(videoID string) (bool, string) {
	var foundPath string

	filepath.Walk(s.outputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == fmt.Sprintf("video_%s.json", videoID) {
			foundPath = filepath.Dir(path)
			return filepath.SkipAll
		}
		return nil
	})

	return foundPath != "", foundPath
}

// checkSubtitlesExist checks if subtitles directory exists for a video in the given folder
func (s *Scrapper) checkSubtitlesExist(folder string) bool {
	subsDir := filepath.Join(folder, "subs")
	if _, err := os.Stat(subsDir); !os.IsNotExist(err) {
		// Check if there's at least one subtitle file
		entries, err := os.ReadDir(subsDir)
		if err == nil && len(entries) > 0 {
			return true
		}
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

func (s *Scrapper) Scrape(maxPages int) (int, []error) {
	videosDownloaded := 0
	errs := make([]error, 0)

	page := 0
	for {
		// Check if we've reached the max pages limit (0 means unlimited)
		if maxPages > 0 && page > maxPages {
			break
		}

		links, err := s.ScrapePage(page)
		if errors.Is(err, ErrPageNotFound) || errors.Is(err, ErrForbidden) {
			break
		}

		if err != nil {
			errs = append(errs, fmt.Errorf("error finding links on page %d: %w", page, err))
			page++
			continue
		}

		for _, link := range links {
			// Check if video already exists before fetching metadata
			exists, existingFolder := s.checkVideoExistsByID(link.ID)

			if exists {
				// Video metadata exists, but check if subtitles are missing
				if !s.checkSubtitlesExist(existingFolder) {
					// Need to download subtitles - fetch metadata for that
					meta, err := s.DownloadVideoMeta(link.ID)
					if err != nil {
						errs = append(errs, fmt.Errorf("Error downloading video metadata for %s: %w", link.ID, err))
						continue
					}

					if s.verbose {
						fmt.Printf("Video exists but subtitles missing, downloading subtitles: %s (ID: %s)\n", meta.LongTitle, link.ID)
					}

					err = s.DownloadSubtitles(meta, existingFolder)
					if err != nil {
						errs = append(errs, fmt.Errorf("Error downloading subtitles for %s: %w", link.ID, err))
					}
				} else {
					if s.verbose {
						fmt.Printf("Already downloaded, ignoring video: (ID: %s)\n", link.ID)
					}
				}
				continue
			}

			// Video doesn't exist, download everything
			meta, err := s.DownloadVideoMeta(link.ID)
			if err != nil {
				errs = append(errs, fmt.Errorf("Error downloading video metadata for %s: %w", link.ID, err))
				continue
			}

			folder, err := s.folderForVideo(meta)
			if err != nil {
				errs = append(errs, fmt.Errorf("Error creating folder for %s: %w", link.ID, err))
				continue
			}
			if err := os.MkdirAll(folder, 0755); err != nil {
				errs = append(errs, fmt.Errorf("Error creating folder for %s: %w", link.ID, err))
				continue
			}

			err = s.SaveVideoToFile(meta, folder)
			if err != nil {
				errs = append(errs, fmt.Errorf("Error saving video metadata for %s: %w", link.ID, err))
				continue
			}

			err = s.DownloadSubtitles(meta, folder)
			if err != nil {
				errs = append(errs, fmt.Errorf("Error downloading subtitles for %s: %w", link.ID, err))
			}

			err = s.updateFolderTime(meta, folder)
			if err != nil {
				errs = append(errs, fmt.Errorf("Error updating folder time for %s: %w", link.ID, err))
			}

			fmt.Printf("Downloaded video %s\n", meta.LongTitle)
			videosDownloaded++
		}

		page++
	}

	return videosDownloaded, errs
}

type VideoInfo struct {
	URL string
	ID  string
}

type Scrapper struct {
	Program    string
	client     *http.Client
	outputPath string
	verbose    bool
}

type Option func(*Scrapper)

func WithOutputPath(path string) Option {
	return func(s *Scrapper) {
		s.outputPath = path
	}
}

func WithVerbose(verbose bool) Option {
	return func(s *Scrapper) {
		s.verbose = verbose
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

var ErrPageNotFound = errors.New("page not found")
var ErrForbidden = errors.New("access not allowed")
