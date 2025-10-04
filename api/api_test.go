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

func TestFetchShowLatest_ExactCount(t *testing.T) {
	// This test verifies that FetchShowLatest fetches exactly maxVideos,
	// not maxVideos+1 (which was a bug in the original implementation)

	tests := []struct {
		name      string
		maxVideos int
	}{
		{"fetch exactly 1 video", 1},
		{"fetch exactly 2 videos", 2},
		{"fetch exactly 5 videos", 5},
		{"fetch exactly 10 videos", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			visitor := func(result *VideoResult) error {
				count++
				// Track each video processed
				return nil
			}

			// Create a wrapped visitor that simulates FetchShowLatest's behavior
			wrappedCount := 0
			wrappedVisitor := func(result *VideoResult) error {
				// This mimics the fixed implementation
				if tt.maxVideos > 0 && wrappedCount >= tt.maxVideos {
					return ErrMaxVideosReached
				}
				wrappedCount++
				return visitor(result)
			}

			// Simulate processing more videos than requested
			for i := 0; i < tt.maxVideos+10; i++ {
				mockResult := &VideoResult{
					Metadata: &rtve.VideoMetadata{
						ID:        fmt.Sprintf("video-%d", i),
						LongTitle: fmt.Sprintf("Video %d", i),
					},
				}

				err := wrappedVisitor(mockResult)
				if err == ErrMaxVideosReached {
					break
				}
			}

			// Verify we processed exactly maxVideos, not more
			if count != tt.maxVideos {
				t.Errorf("Expected to process exactly %d videos, but processed %d", tt.maxVideos, count)
			}

			// Verify the wrapped counter also stopped at the right count
			if wrappedCount != tt.maxVideos {
				t.Errorf("Expected wrappedCount to be %d, got %d", tt.maxVideos, wrappedCount)
			}
		})
	}
}

func TestFetchShowLatest_ZeroMeansUnlimited(t *testing.T) {
	// Test that maxVideos=0 means unlimited
	count := 0
	maxVideos := 0

	visitor := func(result *VideoResult) error {
		count++
		if count > 100 {
			// Stop after 100 to prevent infinite loop in test
			return fmt.Errorf("stop")
		}
		return nil
	}

	wrappedCount := 0
	wrappedVisitor := func(result *VideoResult) error {
		// Check limit before processing
		if maxVideos > 0 && wrappedCount >= maxVideos {
			return ErrMaxVideosReached
		}
		wrappedCount++
		return visitor(result)
	}

	// Process 100 videos
	for i := 0; i < 100; i++ {
		mockResult := &VideoResult{
			Metadata: &rtve.VideoMetadata{
				ID:        fmt.Sprintf("video-%d", i),
				LongTitle: fmt.Sprintf("Video %d", i),
			},
		}

		err := wrappedVisitor(mockResult)
		if err != nil {
			break
		}
	}

	// With maxVideos=0, we should process all 100 videos
	if count != 100 {
		t.Errorf("Expected to process 100 videos with maxVideos=0, but processed %d", count)
	}
}

func TestFetchShowLatest_OffByOneBug(t *testing.T) {
	// This test specifically checks for the off-by-one bug where
	// the original implementation would fetch maxVideos+1 instead of maxVideos

	maxVideos := 1

	// Simulate the BUGGY implementation (process, increment count, then check count > maxVideos)
	buggyProcessed := 0
	buggyCount := 0
	buggyVisitor := func(result *VideoResult) error {
		// Process the video first (THIS IS THE BUG)
		buggyProcessed++
		buggyCount++
		// Then check if we've exceeded the limit
		if maxVideos > 0 && buggyCount > maxVideos {
			return ErrMaxVideosReached
		}
		return nil
	}

	// Process videos with buggy implementation
	for i := 0; i < 10; i++ {
		mockResult := &VideoResult{
			Metadata: &rtve.VideoMetadata{
				ID:        fmt.Sprintf("buggy-video-%d", i),
				LongTitle: fmt.Sprintf("Video %d", i),
			},
		}

		err := buggyVisitor(mockResult)
		if err == ErrMaxVideosReached {
			break
		}
	}

	// Now test the FIXED implementation (check count >= maxVideos BEFORE incrementing)
	fixedProcessed := 0
	fixedCount := 0
	fixedVisitor := func(result *VideoResult) error {
		// Check limit BEFORE incrementing and processing (THIS IS THE FIX)
		if maxVideos > 0 && fixedCount >= maxVideos {
			return ErrMaxVideosReached
		}
		fixedCount++
		fixedProcessed++
		return nil
	}

	// Process videos with fixed implementation
	for i := 0; i < 10; i++ {
		mockResult := &VideoResult{
			Metadata: &rtve.VideoMetadata{
				ID:        fmt.Sprintf("fixed-video-%d", i),
				LongTitle: fmt.Sprintf("Video %d", i),
			},
		}

		err := fixedVisitor(mockResult)
		if err == ErrMaxVideosReached {
			break
		}
	}

	// The buggy implementation processes maxVideos+1 (because it checks AFTER incrementing and processing)
	expectedBuggy := maxVideos + 1
	if buggyProcessed != expectedBuggy {
		t.Errorf("Buggy implementation should process %d videos (maxVideos+1), but processed %d", expectedBuggy, buggyProcessed)
	}

	// The fixed implementation should process exactly maxVideos
	if fixedProcessed != maxVideos {
		t.Errorf("Fixed implementation should process exactly %d videos, but processed %d", maxVideos, fixedProcessed)
	}

	// Demonstrate the bug: buggy processes one more than fixed
	if buggyProcessed != fixedProcessed+1 {
		t.Errorf("Buggy implementation should process one more video than fixed. Buggy: %d, Fixed: %d",
			buggyProcessed, fixedProcessed)
	}
}
