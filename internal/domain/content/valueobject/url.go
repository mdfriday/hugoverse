package valueobject

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

func extractTypeAndID(urlString string) (string, error) {
	// Parse the URL
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}

	// Get the query parameters
	queryParams := parsedURL.Query()

	// Extract the 'type' and 'id' parameters
	postType := queryParams.Get("type")
	postID := queryParams.Get("id")

	// Combine them into the desired format
	result := fmt.Sprintf("%s:%s", postType, postID)

	return result, nil
}

const apiUploadPrefix = "/api/uploads/"

func getAssetAbsPath(apiPath string, uploadDir string) (string, error) {
	if !strings.HasPrefix(apiPath, apiUploadPrefix) {
		return "", fmt.Errorf("path not contain %s", apiUploadPrefix)
	}

	return filepath.Join(uploadDir, apiPath[len(apiUploadPrefix):]), nil
}
