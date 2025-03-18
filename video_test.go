package rtve

import (
	"os"
	"testing"
)

func TestParseMetadata(t *testing.T) {
	// Read the test JSON file
	data, err := os.ReadFile("fixtures/video.json")
	if err != nil {
		t.Fatalf("Failed to read test fixture: %v", err)
	}

	// Parse the metadata
	m := &VideoMetadata{}
	metadata, err := m, m.Parse(string(data))
	if err != nil {
		t.Fatalf("Failed to parse metadata: %v", err)
	}

	// Check that the parsed metadata matches the expected values
	expectedValues := map[string]string{
		"URI":             "https://www.rtve.es/api/videos/16492499",
		"HTMLUrl":         "https://www.rtve.es/play/videos/telediario-2/14-03-25/16492499/",
		"ID":              "16492499",
		"LongTitle":       "Telediario - 21 horas - 14/03/25",
		"PublicationDate": "14-03-2025 21:00:00",
	}

	// Check each field against the expected value
	if metadata.URI != expectedValues["URI"] {
		t.Errorf("Expected URI to be %s, got %s", expectedValues["URI"], metadata.URI)
	}
	if metadata.HTMLUrl != expectedValues["HTMLUrl"] {
		t.Errorf("Expected HTMLUrl to be %s, got %s", expectedValues["HTMLUrl"], metadata.HTMLUrl)
	}
	if metadata.ID != expectedValues["ID"] {
		t.Errorf("Expected ID to be %s, got %s", expectedValues["ID"], metadata.ID)
	}
	if metadata.LongTitle != expectedValues["LongTitle"] {
		t.Errorf("Expected LongTitle to be %s, got %s", expectedValues["LongTitle"], metadata.LongTitle)
	}
	if metadata.PublicationDate != expectedValues["PublicationDate"] {
		t.Errorf("Expected PublicationDate to be %s, got %s", expectedValues["PublicationDate"], metadata.PublicationDate)
	}
}

func TestParseMetadataEmptyResponse(t *testing.T) {
	// Test with empty items array
	emptyJSON := `{"page":{"items":[],"number":1,"size":1,"offset":0,"total":0,"totalPages":0,"numElements":0}}`

	m := &VideoMetadata{}
	err := m.Parse(string(emptyJSON))
	if err == nil {
		t.Error("Expected error for empty items array, got nil")
	}
}

func TestParseMetadataMalformedJSON(t *testing.T) {
	// Test with malformed JSON
	malformedJSON := `{"page":{"items":[{"uri":"https://example.com"]}`

	m := &VideoMetadata{}
	err := m.Parse(malformedJSON)
	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	}
}
