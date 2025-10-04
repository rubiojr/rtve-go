package api

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// Integration tests that make real HTTP calls to RTVE's API.
// These tests use very limited date ranges (only 2-3 days) to avoid
// downloading large amounts of data.

func TestIntegrationFetchShowValidation(t *testing.T) {
	// Use recent dates, only 1 day range
	now := time.Now()
	start := now.AddDate(0, 0, -2) // 2 days ago
	end := now.AddDate(0, 0, -1)   // 1 day ago

	tests := []struct {
		name        string
		showID      string
		startDate   time.Time
		endDate     time.Time
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid show ID",
			showID:      "non-existent-show",
			startDate:   start,
			endDate:     end,
			expectError: true,
			errorMsg:    "invalid show ID",
		},
		{
			name:        "end date before start date",
			showID:      "telediario-1",
			startDate:   end,
			endDate:     start,
			expectError: true,
			errorMsg:    "end date",
		},
		{
			name:        "valid telediario-1",
			showID:      "telediario-1",
			startDate:   start,
			endDate:     end,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visitor := func(result *VideoResult) error {
				return nil
			}

			stats, err := FetchShow(tt.showID, tt.startDate, tt.endDate, visitor)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if stats == nil {
					t.Error("Expected stats to be non-nil")
				}
			}
		})
	}
}

func TestIntegrationFetchShowLatest(t *testing.T) {
	// Only fetch 1-2 videos to keep test fast
	tests := []struct {
		name      string
		showID    string
		maxVideos int
	}{
		{
			name:      "fetch 1 video",
			showID:    "telediario-1",
			maxVideos: 1,
		},
		{
			name:      "fetch 2 videos",
			showID:    "telediario-1",
			maxVideos: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			visitor := func(result *VideoResult) error {
				count++
				if result.Metadata == nil {
					t.Error("Metadata should not be nil")
				}
				return nil
			}

			stats, err := FetchShowLatest(tt.showID, tt.maxVideos, visitor)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if stats == nil {
				t.Fatal("Stats should not be nil")
			}

			// If maxVideos is set and we found that many, verify we stopped at the limit
			if tt.maxVideos > 0 && count > tt.maxVideos {
				t.Errorf("Expected at most %d videos, got %d", tt.maxVideos, count)
			}

			t.Logf("Fetched %d videos (max: %d)", count, tt.maxVideos)
		})
	}
}

func TestIntegrationDateRangeFiltering(t *testing.T) {
	// Test with a very narrow date range - only 1 day
	now := time.Now()
	start := now.AddDate(0, 0, -2)
	end := now.AddDate(0, 0, -1)

	visitor := func(result *VideoResult) error {
		// Parse the publication date
		const rtveLayout = "02-01-2006 15:04:05"
		pubDate, err := time.Parse(rtveLayout, result.Metadata.PublicationDate)
		if err != nil {
			return fmt.Errorf("failed to parse publication date: %w", err)
		}

		// Verify it's within range
		if pubDate.Before(start) || pubDate.After(end) {
			t.Errorf("Video published at %s is outside range [%s, %s]",
				pubDate.Format(time.RFC3339),
				start.Format(time.RFC3339),
				end.Format(time.RFC3339))
		}

		return nil
	}

	stats, err := FetchShow("telediario-1", start, end, visitor)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// It's OK if no videos are found for this specific date
	t.Logf("Found %d videos in date range", stats.VideosProcessed)
}

func TestIntegrationStatsAccuracy(t *testing.T) {
	// Only test with 2 days to keep it fast
	now := time.Now()
	start := now.AddDate(0, 0, -2)
	end := now.AddDate(0, 0, -1)

	videoCount := 0
	visitor := func(result *VideoResult) error {
		videoCount++
		// Stop after 3 videos to keep test fast
		if videoCount >= 3 {
			return fmt.Errorf("reached test limit")
		}
		return nil
	}

	stats, err := FetchShow("telediario-1", start, end, visitor)
	// Might error if we hit the limit
	if err != nil && !contains(err.Error(), "reached test limit") && !errors.Is(err, ErrMaxVideosReached) {
		t.Fatalf("Unexpected error: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	// Verify stats match actual count
	if stats.VideosProcessed != videoCount {
		t.Errorf("Stats.VideosProcessed (%d) doesn't match actual count (%d)",
			stats.VideosProcessed, videoCount)
	}

	// Pages scraped should be at least 1 if we found any videos
	if videoCount > 0 && stats.PagesScraped == 0 {
		t.Error("PagesScraped should be > 0 when videos are found")
	}

	t.Logf("Processed %d videos across %d pages with %d errors",
		stats.VideosProcessed, stats.PagesScraped, stats.ErrorCount)
}

func TestIntegrationSubtitlesInResult(t *testing.T) {
	// Fetch only 1 video and check if subtitles are included
	foundVideo := false
	visitor := func(result *VideoResult) error {
		foundVideo = true

		// Check metadata
		if result.Metadata == nil {
			t.Error("Metadata should not be nil")
		}

		// Subtitles might be nil (not available) or populated
		if result.Subtitles != nil {
			t.Logf("Video %s has %d subtitle tracks", result.Metadata.ID, len(result.Subtitles.Subtitles))
		} else if result.SubtitlesError != nil {
			t.Logf("Subtitle error for video %s: %v", result.Metadata.ID, result.SubtitlesError)
		} else {
			t.Log("No subtitles available for this video")
		}

		// Don't return error, just let it process normally
		return nil
	}

	stats, err := FetchShowLatest("telediario-1", 1, visitor)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	if !foundVideo {
		t.Error("Should have found at least one video")
	}

	if stats.VideosProcessed < 1 {
		t.Error("Should have processed at least one video")
	}
}

func TestIntegrationVisitorErrorHandling(t *testing.T) {
	// Test that visitor errors stop the fetching process
	callCount := 0
	visitor := func(result *VideoResult) error {
		callCount++
		if callCount >= 2 {
			return fmt.Errorf("stop processing")
		}
		return nil
	}

	// Fetch only latest 5
	stats, err := FetchShowLatest("telediario-1", 5, visitor)

	// We expect an error since the visitor returned one (if we found >= 2 videos)
	if err == nil {
		// If no error, it means we didn't find enough videos for the test
		t.Log("Did not find enough videos to trigger visitor error")
	} else {
		// Verify the error is from the visitor
		if !contains(err.Error(), "visitor function returned error") {
			t.Errorf("Expected visitor error, got: %v", err)
		}
		// Stats should still be returned even with error
		if stats == nil {
			t.Error("Stats should be returned even when visitor errors")
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIntegrationFetchLatestAllShows(t *testing.T) {
	// Hardcoded list of current available shows
	shows := []string{"telediario-1", "telediario-2", "telediario-matinal", "informe-semanal"}

	for _, showID := range shows {
		t.Run(showID, func(t *testing.T) {
			foundVideo := false
			visitor := func(result *VideoResult) error {
				foundVideo = true

				// Basic validation
				if result.Metadata == nil {
					t.Error("Metadata should not be nil")
				}

				t.Logf("Found latest video: %s (ID: %s)", result.Metadata.LongTitle, result.Metadata.ID)

				return nil
			}

			// Fetch only 1 latest video per show
			stats, err := FetchShowLatest(showID, 1, visitor)
			if err != nil {
				t.Errorf("Error fetching latest from %s: %v", showID, err)
				return
			}

			if stats == nil {
				t.Error("Stats should not be nil")
				return
			}

			if !foundVideo {
				t.Errorf("Should have found at least one video for %s", showID)
			}

			if stats.VideosProcessed < 1 {
				t.Errorf("Should have processed at least one video for %s, got %d", showID, stats.VideosProcessed)
			}
		})
	}
}
