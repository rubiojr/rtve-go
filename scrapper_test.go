package rtve

import (
	"os"
	"testing"
)

func TestScrape(t *testing.T) {
	data, err := os.ReadFile("fixtures/show.html")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	scraper := NewScrapper("telediario-2")

	videos, err := scraper.scrape(string(data))
	if err != nil {
		t.Fatalf("Failed to scrape HTML: %v", err)
	}

	expectedCount := 20
	if len(videos) != expectedCount {
		t.Errorf("Expected %d videos, got %d", expectedCount, len(videos))
	}

	found := false
	expectedID := "16492499"
	for _, video := range videos {
		if video.ID == expectedID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find video with ID %s, but it was not found", expectedID)
	}

	// Test the URL format
	for _, video := range videos {
		if video.URL == "" {
			t.Errorf("Video URL should not be empty")
		}

		// Check URL format
		expectedURLPrefix := "https://www.rtve.es/play/videos/telediario-2/"
		if len(video.URL) < len(expectedURLPrefix) || video.URL[:len(expectedURLPrefix)] != expectedURLPrefix {
			t.Errorf("Video URL %s does not have expected prefix %s", video.URL, expectedURLPrefix)
		}
	}

	scraperInforme := NewScrapper("informe-semanal")
	_, err = scraperInforme.scrape(string(data))
	if err != nil {
		t.Errorf("Failed to scrape HTML with different show: %v", err)
	}
}
