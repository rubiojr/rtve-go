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

func (s *Scrapper) FetchSubtitles(id string) (*SubtitleResponse, error) {
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

// DownloadSubtitles downloads all available subtitles for a given video ID and saves them to the specified directory
func (s *Scrapper) DownloadSubtitles(meta *VideoMetadata, outputDir string) error {
	outputDir = filepath.Join(outputDir, "subs")

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Fetch subtitle information
	subtitles, err := s.FetchSubtitles(meta.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch subtitles: %v", err)
	}

	// Check if there are any subtitles
	if len(subtitles.Page.Items) == 0 {
		return fmt.Errorf("no subtitles found for video ID: %s", meta.ID)
	}

	// Create HTTP client for downloading
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for _, item := range subtitles.Page.Items {
		// Create a filename based on video ID and language
		filename := fmt.Sprintf("%s_%s.vtt", meta.ID, item.Lang)
		outputPath := filepath.Join(outputDir, filename)

		// Download the subtitle file
		req, err := http.NewRequest("GET", item.Src, nil)
		if err != nil {
			fmt.Printf("Error creating request for %s: %v\n", item.Lang, err)
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error downloading subtitle for %s: %v\n", item.Lang, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			fmt.Printf("Error: received status code %d for %s\n", resp.StatusCode, item.Lang)
			continue
		}

		// Create output file
		out, err := os.Create(outputPath)
		if err != nil {
			resp.Body.Close()
			fmt.Printf("Error creating output file for %s: %v\n", item.Lang, err)
			continue
		}

		// Copy data to file
		_, err = io.Copy(out, resp.Body)
		resp.Body.Close()
		out.Close()

		if err != nil {
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
