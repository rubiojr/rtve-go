package rtve

import (
	"strings"
	"testing"
)

// TestRealWorldHTMLScenarios tests the scraper against realistic HTML scenarios
// that would have exposed the original regex bug
func TestRealWorldHTMLScenarios(t *testing.T) {
	tests := []struct {
		name        string
		show        string
		htmlContent string
		expectedIDs []string
		description string
	}{
		{
			name: "URLs with HTML attributes after them",
			show: "telediario-1",
			htmlContent: `
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/" title="Telediario - 15 horas - 03/10/25">Video 1</a>
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-02-10-25/16754110/" class="video-link" data-id="09">Video 2</a>
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-01-10-25/16752490/" onclick="track('video')">Video 3</a>
			`,
			expectedIDs: []string{"16755959", "16754110", "16752490"},
			description: "This would have caused the original bug - extracting '09' instead of proper IDs",
		},
		{
			name: "URLs followed by numbers in attributes",
			show: "telediario-2",
			htmlContent: `
				<div data-video-id="10">
					<a href="https://www.rtve.es/play/videos/telediario-2/21-horas-17-03-25/16495457/">Video</a>
				</div>
				<div data-order="09">
					<a href="https://www.rtve.es/play/videos/telediario-2/14-03-25/16492499/">Video 2</a>
				</div>
			`,
			expectedIDs: []string{"16495457", "16492499"},
			description: "Numbers in nearby attributes should not be extracted as video IDs",
		},
		{
			name: "Mixed content with many slashes",
			show: "telediario-1",
			htmlContent: `
				<span>Date: 03/10/25</span>
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/" title="Title with / slashes / 09 / 10">Video</a>
				<p>Path: /some/path/09/10/11</p>
			`,
			expectedIDs: []string{"16755959"},
			description: "Various slash-separated content should not confuse the ID extraction",
		},
		{
			name: "Real-world HTML from RTVE pages",
			show: "telediario-1",
			htmlContent: `
<li class="elem_nH">
    <div class="cellBox" data-idasset=16755959>
        <div class="mod video_mod">
            <a class="goto_media" href="https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/" title="Empezar a ver">
                <span class="icon progressBar play">
                    <span class="hour">00:35:18</span>
                </span>
            </a>
        </div>
    </div>
</li>
<li class="elem_nH">
    <div class="cellBox" data-idasset=16754110>
        <a class="goto_media" href="https://www.rtve.es/play/videos/telediario-1/15-horas-02-10-25/16754110/" title="Empezar a ver">
            <span>Video content</span>
        </a>
    </div>
</li>
			`,
			expectedIDs: []string{"16755959", "16754110"},
			description: "Realistic HTML structure from actual RTVE pages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scraper := NewScrapper(tt.show)
			videos, err := scraper.scrape(tt.htmlContent)

			if err != nil {
				t.Fatalf("Scraping failed: %v", err)
			}

			if len(videos) != len(tt.expectedIDs) {
				t.Errorf("%s: Expected %d videos, got %d", tt.description, len(tt.expectedIDs), len(videos))
				t.Logf("Found video IDs: %v", getVideoIDs(videos))
			}

			// Verify all extracted IDs are valid (numeric, reasonable length)
			for _, video := range videos {
				// Check ID length - RTVE video IDs are typically 7-8 digits
				if len(video.ID) < 6 || len(video.ID) > 10 {
					t.Errorf("Video ID '%s' has invalid length %d (should be 6-10 digits). This indicates the regex bug!",
						video.ID, len(video.ID))
				}

				// Check ID is fully numeric
				for _, char := range video.ID {
					if char < '0' || char > '9' {
						t.Errorf("Video ID '%s' contains non-numeric character '%c'. This indicates the regex bug!",
							video.ID, char)
					}
				}

				// Check it's not a short fragment like "09" or "10" that would indicate the bug
				if len(video.ID) <= 2 {
					t.Errorf("Video ID '%s' is too short (%d digits). This is the exact bug we had - extracting fragments instead of full IDs!",
						video.ID, len(video.ID))
				}
			}

			// Verify expected IDs are found
			foundIDs := make(map[string]bool)
			for _, video := range videos {
				foundIDs[video.ID] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !foundIDs[expectedID] {
					t.Errorf("Expected to find video ID %s, but got: %v", expectedID, getVideoIDs(videos))
				}
			}
		})
	}
}

// TestOriginalBugRegression specifically tests the scenario that caused the original bug
func TestOriginalBugRegression(t *testing.T) {
	// The original bug was caused by the greedy regex pattern matching too much
	// and then the ID extraction getting the wrong token

	htmlThatCausedBug := `
		<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/" title="Some title 09">Link 1</a>
		<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-02-10-25/16754110/" data-pos="10">Link 2</a>
	`

	scraper := NewScrapper("telediario-1")
	videos, err := scraper.scrape(htmlThatCausedBug)

	if err != nil {
		t.Fatalf("Scraping failed: %v", err)
	}

	// Before the fix, this would extract IDs like "09" and "10"
	// After the fix, it should extract "16755959" and "16754110"

	for _, video := range videos {
		// The bug manifested as very short IDs (2 digits)
		if len(video.ID) <= 2 {
			t.Fatalf("REGRESSION: The original bug is back! Extracted ID '%s' which is only %d digits. "+
				"Should be a full 8-digit video ID like '16755959'", video.ID, len(video.ID))
		}

		// Verify it's one of the expected IDs
		if video.ID != "16755959" && video.ID != "16754110" {
			t.Errorf("Extracted unexpected video ID: %s", video.ID)
		}
	}

	// Should find exactly 2 videos
	if len(videos) != 2 {
		t.Errorf("Expected 2 videos, got %d", len(videos))
	}
}

// TestIDExtractionLogic tests the core logic of extracting IDs from URLs
func TestIDExtractionLogic(t *testing.T) {
	testCases := []struct {
		url        string
		expectedID string
	}{
		{
			url:        "https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/",
			expectedID: "16755959",
		},
		{
			url:        "https://www.rtve.es/play/videos/telediario-2/telediario-21-horas-05-03-25/16478386/",
			expectedID: "16478386",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedID, func(t *testing.T) {
			// Simulate the ID extraction logic from scrapper.go
			link := tc.url
			if strings.HasSuffix(link, "/") {
				link = link[:len(link)-1]
			}
			tokens := strings.Split(link, "/")
			actualID := tokens[len(tokens)-1]

			if actualID != tc.expectedID {
				t.Errorf("ID extraction failed: expected %s, got %s", tc.expectedID, actualID)
			}

			// Additional validation
			if len(actualID) < 6 {
				t.Errorf("Extracted ID %s is suspiciously short - possible bug", actualID)
			}
		})
	}
}

// TestRegexDoesNotMatchMalformedURLs ensures the regex is strict enough
func TestRegexDoesNotMatchMalformedURLs(t *testing.T) {
	invalidURLs := []struct {
		show string
		url  string
		desc string
	}{
		{
			show: "telediario-1",
			url:  "https://www.rtve.es/play/videos/telediario-1/",
			desc: "URL without video ID",
		},
		{
			show: "telediario-1",
			url:  "https://www.rtve.es/play/videos/telediario-1/title/",
			desc: "URL with title but no ID",
		},
		{
			show: "telediario-1",
			url:  "https://www.example.com/play/videos/telediario-1/title/12345678/",
			desc: "Wrong domain",
		},
		{
			show: "telediario-1",
			url:  "https://www.rtve.es/play/videos/telediario-2/title/12345678/",
			desc: "Wrong show",
		},
	}

	for _, tc := range invalidURLs {
		t.Run(tc.desc, func(t *testing.T) {
			scraper := NewScrapper(tc.show)
			videos, err := scraper.scrape(tc.url)

			if err != nil {
				// Error is acceptable for malformed content
				return
			}

			if len(videos) > 0 {
				t.Errorf("Regex matched invalid URL '%s': %s. Got videos: %v",
					tc.url, tc.desc, getVideoIDs(videos))
			}
		})
	}
}
