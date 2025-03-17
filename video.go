package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// VideoMetadata represents essential metadata from a video
type VideoMetadata struct {
	URI             string `json:"uri"`
	HTMLUrl         string `json:"htmlUrl"`
	ID              string `json:"id"`
	LongTitle       string `json:"longTitle"`
	PublicationDate string `json:"publicationDate"`
}

// VideoPage represents the page of video items
type VideoPage struct {
	Items       []map[string]interface{} `json:"items"`
	Number      int                      `json:"number"`
	Size        int                      `json:"size"`
	Offset      int                      `json:"offset"`
	Total       int                      `json:"total"`
	TotalPages  int                      `json:"totalPages"`
	NumElements int                      `json:"numElements"`
}

// VideoResponse represents the top-level JSON response
type VideoResponse struct {
	Page VideoPage `json:"page"`
}

// DownloadVideoMeta fetches and parses video metadata for a given video ID
func (s *Scrapper) DownloadVideoMeta(videoID string) (*VideoMetadata, error) {
	url := fmt.Sprintf(ApiURL, videoID)

	body, err := s.get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching video metadata: %v", err)
	}

	// Parse the JSON response
	var videoResp VideoResponse
	if err := json.Unmarshal([]byte(body), &videoResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	// Check if we have any items
	if len(videoResp.Page.Items) == 0 {
		return nil, fmt.Errorf("no video metadata found for ID: %s", videoID)
	}

	// Extract the required fields from the first item
	item := videoResp.Page.Items[0]

	metadata := &VideoMetadata{
		URI:             getStringValue(item, "uri"),
		HTMLUrl:         getStringValue(item, "htmlUrl"),
		ID:              getStringValue(item, "id"),
		LongTitle:       getStringValue(item, "longTitle"),
		PublicationDate: getStringValue(item, "publicationDate"),
	}

	return metadata, nil
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

// Helper function to safely extract string values from a map
func getStringValue(item map[string]interface{}, key string) string {
	if val, ok := item[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}
