// Package api provides a high-level interface for fetching video metadata and subtitles
// from RTVE (Radio Televisión Española) on-demand streaming service.
//
// The package offers a simple API for retrieving TV show episodes with their metadata
// and subtitles within a specified date range. Results are processed through a visitor
// function, allowing for flexible handling of each video as it's fetched.
//
// Example usage:
//
//	import (
//		"fmt"
//		"time"
//		"github.com/rubiojr/rtve-go/api"
//	)
//
//	func main() {
//		start := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)
//		end := time.Date(2025, 10, 3, 23, 59, 59, 0, time.UTC)
//
//		visitor := func(result *api.VideoResult) error {
//			fmt.Printf("Found: %s (ID: %s)\n", result.Metadata.LongTitle, result.Metadata.ID)
//			if result.Subtitles != nil {
//				fmt.Printf("  Subtitles available: %d entries\n", len(result.Subtitles.Subtitles))
//			}
//			return nil
//		}
//
//		stats, err := api.FetchShow("telediario-1", start, end, visitor)
//		if err != nil {
//			panic(err)
//		}
//
//		fmt.Printf("Fetched %d videos with %d errors\n", stats.VideosProcessed, stats.ErrorCount)
//	}
package api

import (
	"errors"
	"fmt"
	"time"

	rtve "github.com/rubiojr/rtve-go"
)

// ErrMaxVideosReached is returned when the maximum number of videos has been fetched.
var ErrMaxVideosReached = errors.New("maximum video count reached")

// VideoResult represents the complete data for a single video,
// including its metadata and subtitles (if available).
type VideoResult struct {
	// Metadata contains video information such as title, publication date,
	// duration, and URLs.
	Metadata *rtve.VideoMetadata

	// Subtitles contains the subtitle data for the video.
	// This field will be nil if subtitles are not available or
	// if there was an error fetching them.
	Subtitles *rtve.Subtitles

	// SubtitlesError contains any error that occurred while fetching subtitles.
	// If this is non-nil, the Subtitles field will be nil.
	SubtitlesError error
}

// VisitorFunc is a function type that processes each video result as it's fetched.
// The function receives a VideoResult containing the video's metadata and subtitles.
//
// If the visitor function returns an error, the fetching process will stop immediately
// and return that error. Return nil to continue processing remaining videos.
//
// Example:
//
//	visitor := func(result *api.VideoResult) error {
//		// Process the video
//		fmt.Printf("Processing: %s\n", result.Metadata.LongTitle)
//
//		// Save to database, write to file, etc.
//		if err := saveVideo(result); err != nil {
//			return err // Stop processing on critical error
//		}
//
//		return nil // Continue to next video
//	}
type VisitorFunc func(result *VideoResult) error

// FetchStats contains statistics about the fetch operation.
type FetchStats struct {
	// VideosProcessed is the total number of videos successfully processed
	// by the visitor function.
	VideosProcessed int

	// ErrorCount is the number of non-fatal errors encountered during fetching.
	// These are errors that didn't stop the entire process (e.g., subtitle fetch failures).
	ErrorCount int

	// Errors contains all non-fatal errors encountered during the fetch operation.
	// Fatal errors that stop processing are returned as the function's error return value.
	Errors []error

	// PagesScraped is the number of web pages that were scraped to find videos.
	PagesScraped int
}

// FetchShow fetches video metadata and subtitles for a specific RTVE show
// within the given date range. Each video found is processed by the visitor function.
//
// Parameters:
//
//   - showID: The identifier of the show to fetch. Valid values include:
//     "telediario-1", "telediario-2", "telediario-matinal", "informe-semanal".
//     Use rtve.ListShows() to get all available shows.
//
//   - startDate: The start of the date range (inclusive). Videos published before
//     this date will be excluded.
//
//   - endDate: The end of the date range (inclusive). Videos published after
//     this date will be excluded.
//
//   - visitor: A function that will be called for each video found. The function
//     receives a VideoResult containing the video's metadata and subtitles.
//     If the visitor returns an error, fetching stops immediately.
//
// Returns:
//
//   - *FetchStats: Statistics about the fetch operation, including the number of
//     videos processed and any non-fatal errors encountered.
//
//   - error: A fatal error that stopped the fetching process, such as invalid
//     show ID, network errors, or an error returned by the visitor function.
//     Returns nil if the operation completed successfully.
//
// The function automatically handles pagination, iterating through multiple pages
// of results until it reaches videos outside the date range or runs out of content.
//
// Example:
//
//	start := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
//	end := time.Date(2025, 9, 30, 23, 59, 59, 0, time.UTC)
//
//	visitor := func(result *api.VideoResult) error {
//		fmt.Printf("Title: %s\n", result.Metadata.LongTitle)
//		fmt.Printf("Published: %s\n", result.Metadata.PublicationDate)
//		if result.Subtitles != nil {
//			fmt.Printf("Subtitle entries: %d\n", len(result.Subtitles.Subtitles))
//		}
//		return nil
//	}
//
//	stats, err := api.FetchShow("telediario-1", start, end, visitor)
//	if err != nil {
//		log.Fatalf("Failed to fetch show: %v", err)
//	}
//
//	fmt.Printf("Successfully processed %d videos\n", stats.VideosProcessed)
func FetchShow(showID string, startDate, endDate time.Time, visitor VisitorFunc) (*FetchStats, error) {
	// Validate show ID
	availableShows := rtve.ListShows()
	validShow := false
	for _, show := range availableShows {
		if show == showID {
			validShow = true
			break
		}
	}
	if !validShow {
		return nil, fmt.Errorf("invalid show ID: %s (use rtve.ListShows() to see available shows)", showID)
	}

	// Validate date range
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end date (%s) is before start date (%s)", endDate.Format(time.RFC3339), startDate.Format(time.RFC3339))
	}

	stats := &FetchStats{
		Errors: make([]error, 0),
	}

	scraper := rtve.NewScrapper(showID)

	// The date format used by RTVE
	const rtveLayout = "02-01-2006 15:04:05"

	// Iterate through pages until we're outside the date range
	// or hit an error
	page := 0
	foundVideosInRange := false

	for {
		videos, err := scraper.ScrapePage(page)
		if err != nil {
			// If we've found at least one video in range and now hit an error,
			// we might have just run out of pages - this is OK
			if foundVideosInRange && (err == rtve.ErrPageNotFound || err == rtve.ErrForbidden) {
				break
			}
			// Otherwise, it's a real error
			if err == rtve.ErrPageNotFound || err == rtve.ErrForbidden {
				// No videos found at all - might be valid if date range is in the future
				break
			}
			return stats, fmt.Errorf("error scraping page %d: %w", page, err)
		}

		stats.PagesScraped++

		if len(videos) == 0 {
			// No more videos to process
			break
		}

		videosProcessedThisPage := 0
		allVideosBeforeRange := true

		for _, videoInfo := range videos {
			// Fetch metadata
			metadata, err := scraper.DownloadVideoMeta(videoInfo.ID)
			if err != nil {
				stats.ErrorCount++
				stats.Errors = append(stats.Errors, fmt.Errorf("error fetching metadata for video %s: %w", videoInfo.ID, err))
				continue
			}

			// Parse publication date
			pubDate, err := time.Parse(rtveLayout, metadata.PublicationDate)
			if err != nil {
				stats.ErrorCount++
				stats.Errors = append(stats.Errors, fmt.Errorf("error parsing date for video %s: %w", videoInfo.ID, err))
				continue
			}

			// Check if video is in date range
			if pubDate.Before(startDate) {
				// Video is before our range, continue checking others on this page
				continue
			}

			if pubDate.After(endDate) {
				// Video is after our range, but there might be older videos on this page
				allVideosBeforeRange = false
				continue
			}

			// Video is in range!
			foundVideosInRange = true
			allVideosBeforeRange = false

			// Fetch subtitles
			result := &VideoResult{
				Metadata: metadata,
			}

			subtitles, err := scraper.FetchSubtitles(metadata)
			if err != nil {
				result.SubtitlesError = err
				stats.ErrorCount++
				stats.Errors = append(stats.Errors, fmt.Errorf("error fetching subtitles for video %s: %w", videoInfo.ID, err))
			} else {
				result.Subtitles = subtitles
			}

			// Call visitor function
			if err := visitor(result); err != nil {
				return stats, fmt.Errorf("visitor function returned error for video %s: %w", videoInfo.ID, err)
			}

			stats.VideosProcessed++
			videosProcessedThisPage++
		}

		// If we've found videos in range before, and now all videos on this page
		// are before our start date, we can stop - pages are sorted by date descending
		if foundVideosInRange && allVideosBeforeRange {
			break
		}

		// If we didn't process any videos on this page and we've already found some,
		// we might be past our date range
		if videosProcessedThisPage == 0 && foundVideosInRange {
			// Continue for one more page to be sure, but if the next page also
			// has no results in range, we'll stop
			page++
			videos, err := scraper.ScrapePage(page)
			if err != nil || len(videos) == 0 {
				break
			}
			// Check if any videos on next page are in range
			anyInRange := false
			for _, videoInfo := range videos {
				metadata, err := scraper.DownloadVideoMeta(videoInfo.ID)
				if err != nil {
					continue
				}
				pubDate, err := time.Parse(rtveLayout, metadata.PublicationDate)
				if err != nil {
					continue
				}
				if !pubDate.Before(startDate) && !pubDate.After(endDate) {
					anyInRange = true
					break
				}
			}
			if !anyInRange {
				break
			}
			// If we found some in range, decrement page so the main loop processes it
			page--
		}

		page++
	}

	return stats, nil
}

// FetchShowAll is a convenience function that fetches all available videos for a show
// without date restrictions. It's equivalent to calling FetchShow with a very wide date range.
//
// Parameters:
//   - showID: The identifier of the show to fetch.
//   - visitor: A function that will be called for each video found.
//
// Returns:
//   - *FetchStats: Statistics about the fetch operation.
//   - error: Any fatal error that stopped the fetching process.
//
// Example:
//
//	stats, err := api.FetchShowAll("telediario-1", func(result *api.VideoResult) error {
//		fmt.Printf("Found: %s\n", result.Metadata.LongTitle)
//		return nil
//	})
func FetchShowAll(showID string, visitor VisitorFunc) (*FetchStats, error) {
	// Use a very wide date range
	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Now().Add(24 * time.Hour) // Include today and tomorrow
	return FetchShow(showID, start, end, visitor)
}

// FetchShowLatest fetches the most recent videos for a show, up to maxVideos count.
//
// Parameters:
//   - showID: The identifier of the show to fetch.
//   - maxVideos: Maximum number of videos to fetch. Use 0 for unlimited.
//   - visitor: A function that will be called for each video found.
//
// Returns:
//   - *FetchStats: Statistics about the fetch operation.
//   - error: Any fatal error that stopped the fetching process.
//
// Example:
//
//	// Fetch the 10 most recent episodes
//	stats, err := api.FetchShowLatest("telediario-1", 10, func(result *api.VideoResult) error {
//		fmt.Printf("Recent: %s\n", result.Metadata.LongTitle)
//		return nil
//	})
func FetchShowLatest(showID string, maxVideos int, visitor VisitorFunc) (*FetchStats, error) {
	count := 0
	wrappedVisitor := func(result *VideoResult) error {
		// Check limit before processing
		if maxVideos > 0 && count >= maxVideos {
			// Stop processing by returning a sentinel error
			return ErrMaxVideosReached
		}
		count++
		return visitor(result)
	}

	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Now().Add(24 * time.Hour)

	stats, err := FetchShow(showID, start, end, wrappedVisitor)

	// If we stopped because we reached max videos, that's not an error
	if err != nil && errors.Is(err, ErrMaxVideosReached) {
		return stats, nil
	}

	return stats, err
}

// AvailableShows returns a list of all available show IDs that can be used
// with FetchShow and related functions.
//
// Returns:
//   - []string: A slice of show IDs (e.g., ["telediario-1", "telediario-2", ...])
//
// Example:
//
//	shows := api.AvailableShows()
//	for _, show := range shows {
//		fmt.Printf("Available show: %s\n", show)
//	}
func AvailableShows() []string {
	return rtve.ListShows()
}
