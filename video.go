package rtve

import (
	"encoding/json"
	"fmt"
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
	Items       []VideoMetadata `json:"items"`
	Number      int             `json:"number"`
	Size        int             `json:"size"`
	Offset      int             `json:"offset"`
	Total       int             `json:"total"`
	TotalPages  int             `json:"totalPages"`
	NumElements int             `json:"numElements"`
}

// VideoResponse represents the top-level JSON response
type VideoResponse struct {
	Page VideoPage `json:"page"`
}

func (m *VideoMetadata) Parse(body string) error {
	var videoResp VideoResponse
	if err := json.Unmarshal([]byte(body), &videoResp); err != nil {
		return fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	if len(videoResp.Page.Items) == 0 {
		return fmt.Errorf("no video metadata found")
	}

	*m = videoResp.Page.Items[0]

	return nil
}
