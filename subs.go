package rtve

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SubtitleItem struct {
	Src  string `json:"src"`  // URL of the subtitle file
	Lang string `json:"lang"` // Language code for the subtitle
}

// SubtitlePage represents the page of subtitle items along with pagination info
type SubtitlePage struct {
	Items       []SubtitleItem `json:"items"`       // List of subtitle items
	Number      int            `json:"number"`      // Current page number
	Size        int            `json:"size"`        // Page size
	Offset      int            `json:"offset"`      // Offset in the results
	Total       int            `json:"total"`       // Total number of items
	TotalPages  int            `json:"totalPages"`  // Total number of pages
	NumElements int            `json:"numElements"` // Number of elements in the current page
}

// SubtitleResponse represents the top-level JSON response
type SubtitleResponse struct {
	Page SubtitlePage `json:"page"` // Subtitle page information
}

// Subtitles represents parsed subtitle data for a video
type Subtitles struct {
	// VideoID is the ID of the video these subtitles belong to
	VideoID string
	// Subtitles is a list of available subtitle tracks
	Subtitles []SubtitleItem
}

// FetchSubtitles fetches subtitle metadata for a video and returns a Subtitles object
func (s *Scrapper) FetchSubtitles(meta *VideoMetadata) (*Subtitles, error) {
	url := fmt.Sprintf(SubsURL, meta.ID)

	body, err := s.get(url)
	if err != nil {
		return nil, err
	}

	var subtitleResp SubtitleResponse
	if err := json.Unmarshal([]byte(body), &subtitleResp); err != nil {
		return nil, err
	}

	return &Subtitles{
		VideoID:   meta.ID,
		Subtitles: subtitleResp.Page.Items,
	}, nil
}

func (s *Scrapper) fetchSubtitlesResponse(id string) (*SubtitleResponse, error) {
	url := fmt.Sprintf(SubsURL, id)

	body, err := s.get(url)
	if err != nil {
		return nil, err
	}

	var subtitleResp SubtitleResponse
	if err := json.Unmarshal([]byte(body), &subtitleResp); err != nil {
		return nil, err
	}

	return &subtitleResp, nil
}

// downloadWithRetry downloads a file with retry logic for 5xx errors
func (s *Scrapper) downloadWithRetry(url string, maxRetries int) ([]byte, error) {
	const initialBackoff = 1 * time.Second

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error executing request: %v", err)
		}

		// Retry on 5xx errors
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			resp.Body.Close()
			if attempt < maxRetries {
				backoff := initialBackoff * time.Duration(1<<uint(attempt))
				if s.verbose {
					fmt.Printf("Server error %d downloading subtitle, retrying in %v (attempt %d/%d)...\n", resp.StatusCode, backoff, attempt+1, maxRetries)
				}
				time.Sleep(backoff)
				continue
			}
			return nil, fmt.Errorf("server error after %d retries: status code %d", maxRetries, resp.StatusCode)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %w", err)
		}

		return body, nil
	}

	return nil, fmt.Errorf("unexpected error in retry loop")
}

// DownloadSubtitles downloads all available subtitles for a given video ID and saves them to the specified directory
func (s *Scrapper) DownloadSubtitles(meta *VideoMetadata, outputDir string) error {
	outputDir = filepath.Join(outputDir, "subs")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Fetch subtitle information
	subtitles, err := s.fetchSubtitlesResponse(meta.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch subtitles: %v", err)
	}

	// Check if there are any subtitles
	if len(subtitles.Page.Items) == 0 {
		return fmt.Errorf("no subtitles found for video ID: %s", meta.ID)
	}

	for _, item := range subtitles.Page.Items {
		// Create a filename based on video ID and language
		filename := fmt.Sprintf("%s_%s.vtt", meta.ID, item.Lang)
		outputPath := filepath.Join(outputDir, filename)

		// Download the subtitle file with retries
		content, err := s.downloadWithRetry(item.Src, 3)
		if err != nil {
			fmt.Printf("Error downloading subtitle for %s: %v\n", item.Lang, err)
			continue
		}

		// Write to file
		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			fmt.Printf("Error writing subtitle for %s: %v\n", item.Lang, err)
			continue
		}
	}

	return nil
}

// Helper function to get language name from language code
func GetLanguageName(langCode string) string {
	languages := map[string]string{
		"es": "Spanish",
		"en": "English",
		"ca": "Catalan",
		"eu": "Basque",
		"gl": "Galician",
		// Add more languages as needed
	}

	if name, ok := languages[strings.ToLower(langCode)]; ok {
		return name
	}
	return langCode
}
