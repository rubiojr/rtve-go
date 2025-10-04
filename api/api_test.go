package api

import (
	"fmt"
	"testing"

	rtve "github.com/rubiojr/rtve-go"
)

func TestVideoResultStructure(t *testing.T) {
	// Test that VideoResult has the expected fields
	result := &VideoResult{
		Metadata: &rtve.VideoMetadata{
			ID:        "12345678",
			LongTitle: "Test Video",
		},
		Subtitles: &rtve.Subtitles{
			VideoID:   "12345678",
			Subtitles: []rtve.SubtitleItem{},
		},
		SubtitlesError: nil,
	}

	if result.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
	if result.Metadata.ID != "12345678" {
		t.Errorf("Expected ID '12345678', got '%s'", result.Metadata.ID)
	}
	if result.Subtitles == nil {
		t.Error("Subtitles should not be nil")
	}
	if result.SubtitlesError != nil {
		t.Error("SubtitlesError should be nil")
	}
}

func TestFetchStatsStructure(t *testing.T) {
	stats := &FetchStats{
		VideosProcessed: 5,
		ErrorCount:      2,
		Errors:          []error{fmt.Errorf("error 1"), fmt.Errorf("error 2")},
		PagesScraped:    3,
	}

	if stats.VideosProcessed != 5 {
		t.Errorf("Expected VideosProcessed=5, got %d", stats.VideosProcessed)
	}
	if stats.ErrorCount != 2 {
		t.Errorf("Expected ErrorCount=2, got %d", stats.ErrorCount)
	}
	if len(stats.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(stats.Errors))
	}
	if stats.PagesScraped != 3 {
		t.Errorf("Expected PagesScraped=3, got %d", stats.PagesScraped)
	}
}

func TestVisitorFunctionSignature(t *testing.T) {
	// Test that VisitorFunc has the correct signature
	var visitor VisitorFunc = func(result *VideoResult) error {
		if result == nil {
			return fmt.Errorf("result is nil")
		}
		return nil
	}

	// Test calling the visitor
	testResult := &VideoResult{
		Metadata: &rtve.VideoMetadata{ID: "test"},
	}

	err := visitor(testResult)
	if err != nil {
		t.Errorf("Visitor should not return error for valid result: %v", err)
	}

	// Test with nil result
	err = visitor(nil)
	if err == nil {
		t.Error("Visitor should return error for nil result")
	}
}

func TestAvailableShows(t *testing.T) {
	shows := AvailableShows()

	if len(shows) == 0 {
		t.Error("Expected at least one show to be available")
	}

	// Check that expected shows are present
	expectedShows := []string{"telediario-1", "telediario-2", "telediario-matinal", "informe-semanal"}
	for _, expected := range expectedShows {
		found := false
		for _, show := range shows {
			if show == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected show '%s' not found in available shows: %v", expected, shows)
		}
	}
}

func TestFetchShowValidation(t *testing.T) {
	tests := []struct {
		name        string
		showID      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid show ID",
			showID:      "non-existent-show",
			expectError: true,
			errorMsg:    "invalid show ID",
		},
		{
			name:        "valid telediario-1",
			showID:      "telediario-1",
			expectError: false,
		},
		{
			name:        "valid telediario-2",
			showID:      "telediario-2",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate show ID
			availableShows := AvailableShows()
			validShow := false
			for _, show := range availableShows {
				if show == tt.showID {
					validShow = true
					break
				}
			}

			if tt.expectError {
				if validShow {
					t.Error("Expected invalid show ID to not be in available shows")
				}
			} else {
				if !validShow {
					t.Errorf("Expected show '%s' to be valid", tt.showID)
				}
			}
		})
	}
}
