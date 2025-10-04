package rtve

import (
	"regexp"
	"strings"
	"testing"
)

func TestURLRegexPatterns(t *testing.T) {
	tests := []struct {
		name        string
		show        string
		htmlContent string
		expectedIDs []string
		shouldFail  bool
	}{
		{
			name: "telediario-1 clean URLs",
			show: "telediario-1",
			htmlContent: `
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/">Video 1</a>
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-02-10-25/16754110/">Video 2</a>
			`,
			expectedIDs: []string{"16755959", "16754110"},
		},
		{
			name: "telediario-1 URLs with extra content (real-world scenario)",
			show: "telediario-1",
			htmlContent: `
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/" title="Telediario - 15 horas - 03/10/">Video 1</a>
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-02-10-25/16754110/" class="video-link">Video 2</a>
			`,
			expectedIDs: []string{"16755959", "16754110"},
		},
		{
			name: "telediario-2 URLs",
			show: "telediario-2",
			htmlContent: `
				<a href="https://www.rtve.es/play/videos/telediario-2/21-horas-17-03-25/16495457/">Video 1</a>
				<a href="https://www.rtve.es/play/videos/telediario-2/telediario-21-horas-05-03-25/16478386/">Video 2</a>
			`,
			expectedIDs: []string{"16495457", "16478386"},
		},
		{
			name: "telediario-matinal URLs",
			show: "telediario-matinal",
			htmlContent: `
				<a href="https://www.rtve.es/play/videos/telediario-matinal/09-10-25/16759123/">Video 1</a>
				<a href="https://www.rtve.es/play/videos/telediario-matinal/matinal-08-10-25/16757456/">Video 2</a>
			`,
			expectedIDs: []string{"16759123", "16757456"},
		},
		{
			name: "informe-semanal URLs",
			show: "informe-semanal",
			htmlContent: `
				<a href="https://www.rtve.es/play/videos/informe-semanal/05-07-25/16123456/">Video 1</a>
				<a href="https://www.rtve.es/play/videos/informe-semanal/informe-26-07-25/16234567/">Video 2</a>
			`,
			expectedIDs: []string{"16123456", "16234567"},
		},
		{
			name: "mixed URLs with non-matching content",
			show: "telediario-1",
			htmlContent: `
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/">Valid Video</a>
				<a href="https://www.rtve.es/play/videos/telediario-2/21-horas-02-10-25/16754110/">Wrong show</a>
				<a href="https://www.example.com/some-other-site/">External link</a>
				<a href="https://www.rtve.es/play/videos/telediario-1/another-video/16123456/">Another valid</a>
			`,
			expectedIDs: []string{"16755959", "16123456"},
		},
		{
			name: "URLs that would cause the old bug",
			show: "telediario-1",
			htmlContent: `
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/" title="Some title with / slashes / 09">Video 1</a>
				<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-02-10-25/16754110/" data-id="10" class="link">Video 2</a>
			`,
			expectedIDs: []string{"16755959", "16754110"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scraper := NewScrapper(tt.show)

			videos, err := scraper.scrape(tt.htmlContent)
			if err != nil {
				if !tt.shouldFail {
					t.Errorf("Unexpected error: %v", err)
				}
				return
			}

			if tt.shouldFail {
				t.Errorf("Expected test to fail but it passed")
				return
			}

			// Check we found the expected number of videos
			if len(videos) != len(tt.expectedIDs) {
				t.Errorf("Expected %d videos, got %d", len(tt.expectedIDs), len(videos))
			}

			// Check each expected ID is found
			foundIDs := make(map[string]bool)
			for _, video := range videos {
				foundIDs[video.ID] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !foundIDs[expectedID] {
					t.Errorf("Expected to find video ID %s, but it was not found. Found IDs: %v", expectedID, getVideoIDs(videos))
				}
			}

			// Validate all extracted IDs are numeric and reasonable length
			for _, video := range videos {
				if len(video.ID) < 6 || len(video.ID) > 10 {
					t.Errorf("Video ID %s has unexpected length %d (should be 6-10 digits)", video.ID, len(video.ID))
				}

				// Check ID is all numeric
				for _, char := range video.ID {
					if char < '0' || char > '9' {
						t.Errorf("Video ID %s contains non-numeric character: %c", video.ID, char)
					}
				}
			}
		})
	}
}

func TestRegexPatternsDirectly(t *testing.T) {
	testCases := []struct {
		name        string
		show        string
		testURL     string
		shouldMatch bool
	}{
		{
			name:        "telediario-1 valid URL",
			show:        "telediario-1",
			testURL:     "https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/",
			shouldMatch: true,
		},
		{
			name:        "telediario-1 URL without trailing slash",
			show:        "telediario-1",
			testURL:     "https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959",
			shouldMatch: false,
		},
		{
			name:        "telediario-1 URL with extra path",
			show:        "telediario-1",
			testURL:     "https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/extra",
			shouldMatch: true, // Regex matches the valid pattern within the string; extra content doesn't prevent match
		},
		{
			name:        "telediario-2 valid URL",
			show:        "telediario-2",
			testURL:     "https://www.rtve.es/play/videos/telediario-2/21-horas-17-03-25/16495457/",
			shouldMatch: true,
		},
		{
			name:        "wrong show URL",
			show:        "telediario-1",
			testURL:     "https://www.rtve.es/play/videos/telediario-2/21-horas-17-03-25/16495457/",
			shouldMatch: false,
		},
		{
			name:        "non-RTVE URL",
			show:        "telediario-1",
			testURL:     "https://www.example.com/play/videos/telediario-1/15-horas-03-10-25/16755959/",
			shouldMatch: false,
		},
		{
			name:        "informe-semanal with hyphen in regex",
			show:        "informe-semanal",
			testURL:     "https://www.rtve.es/play/videos/informe-semanal/05-07-25/16123456/",
			shouldMatch: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			show := ShowMap(tc.show)
			if show == nil {
				t.Fatalf("Unknown show: %s", tc.show)
			}

			pattern := regexp.MustCompile(show.Regex)
			matches := pattern.MatchString(tc.testURL)

			if matches != tc.shouldMatch {
				t.Errorf("URL %s with pattern %s: expected match=%t, got match=%t",
					tc.testURL, show.Regex, tc.shouldMatch, matches)
			}
		})
	}
}

func TestVideoIDExtraction(t *testing.T) {
	testCases := []struct {
		name       string
		url        string
		expectedID string
	}{
		{
			name:       "standard telediario-1 URL",
			url:        "https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/",
			expectedID: "16755959",
		},
		{
			name:       "telediario-2 URL with longer title",
			url:        "https://www.rtve.es/play/videos/telediario-2/telediario-21-horas-05-03-25/16478386/",
			expectedID: "16478386",
		},
		{
			name:       "short video ID",
			url:        "https://www.rtve.es/play/videos/telediario-1/test/123456/",
			expectedID: "123456",
		},
		{
			name:       "long video ID",
			url:        "https://www.rtve.es/play/videos/telediario-1/test/1234567890/",
			expectedID: "1234567890",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the ID extraction logic from scrapper.go
			if strings.HasSuffix(tc.url, "/") {
				tc.url = tc.url[:len(tc.url)-1]
			}
			tokens := strings.Split(tc.url, "/")
			actualID := tokens[len(tokens)-1]

			if actualID != tc.expectedID {
				t.Errorf("Expected ID %s, got %s", tc.expectedID, actualID)
			}
		})
	}
}

func TestOldBugScenario(t *testing.T) {
	// This test specifically recreates the scenario that caused the original bug
	htmlWithProblematicURLs := `
		<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/" title="Telediario - 15 horas - 03/10/25">Video 1</a>
		<a href="https://www.rtve.es/play/videos/telediario-1/15-horas-02-10-25/16754110/" class="some-class" data-id="09">Video 2</a>
		<span>Some other content with numbers like 10 and 11</span>
	`

	// Test with old regex pattern (this would have failed)
	oldRegex := `https://www\.rtve\.es/play/videos/telediario-1/.*/`
	oldPattern := regexp.MustCompile(oldRegex)
	oldMatches := oldPattern.FindAllString(htmlWithProblematicURLs, -1)

	// Test with new regex pattern (this should work)
	newRegex := `https://www\.rtve\.es/play/videos/telediario-1/[^/]+/[0-9]+/`
	newPattern := regexp.MustCompile(newRegex)
	newMatches := newPattern.FindAllString(htmlWithProblematicURLs, -1)

	t.Logf("Old regex matches: %v", oldMatches)
	t.Logf("New regex matches: %v", newMatches)

	// With the old regex, we would get malformed URLs
	// Let's verify the new regex gives us clean URLs
	expectedURLs := []string{
		"https://www.rtve.es/play/videos/telediario-1/15-horas-03-10-25/16755959/",
		"https://www.rtve.es/play/videos/telediario-1/15-horas-02-10-25/16754110/",
	}

	if len(newMatches) != len(expectedURLs) {
		t.Errorf("Expected %d matches, got %d", len(expectedURLs), len(newMatches))
	}

	for i, expectedURL := range expectedURLs {
		if i < len(newMatches) && newMatches[i] != expectedURL {
			t.Errorf("Expected URL %s, got %s", expectedURL, newMatches[i])
		}
	}

	// Now test ID extraction from the clean URLs
	for _, url := range newMatches {
		cleanURL := url
		if strings.HasSuffix(cleanURL, "/") {
			cleanURL = cleanURL[:len(cleanURL)-1]
		}
		tokens := strings.Split(cleanURL, "/")
		id := tokens[len(tokens)-1]

		// The ID should be numeric and reasonable length
		if len(id) < 6 {
			t.Errorf("Extracted ID %s is too short (would indicate the old bug)", id)
		}

		// Should be all numeric
		for _, char := range id {
			if char < '0' || char > '9' {
				t.Errorf("Extracted ID %s contains non-numeric characters (would indicate the old bug)", id)
			}
		}
	}
}

// Helper function to get video IDs from VideoInfo slice
func getVideoIDs(videos []*VideoInfo) []string {
	var ids []string
	for _, video := range videos {
		ids = append(ids, video.ID)
	}
	return ids
}

func TestAllShowRegexPatterns(t *testing.T) {
	// Test that all shows have valid regex patterns that work correctly
	shows := []string{"telediario-1", "telediario-2", "telediario-matinal", "informe-semanal"}

	for _, show := range shows {
		t.Run(show, func(t *testing.T) {
			showInfo := ShowMap(show)
			if showInfo == nil {
				t.Fatalf("Show %s not found", show)
			}

			// Test that the regex compiles
			_, err := regexp.Compile(showInfo.Regex)
			if err != nil {
				t.Fatalf("Regex for %s does not compile: %v", show, err)
			}

			// Test with a sample URL for this show
			sampleURL := "https://www.rtve.es/play/videos/" + show + "/test-title/12345678/"
			pattern := regexp.MustCompile(showInfo.Regex)

			if !pattern.MatchString(sampleURL) {
				t.Errorf("Regex for %s should match sample URL %s", show, sampleURL)
			}
		})
	}
}
