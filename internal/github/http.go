package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func getJSON(ctx context.Context, url string, target any) error {
	_, err := getJSONConditional(ctx, url, "", target)
	return err
}

// getJSONConditional performs a GET with optional If-None-Match header for ETag support.
func getJSONConditional(ctx context.Context, reqURL string, etag string, target any) (result conditionalResult, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return conditionalResult{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return conditionalResult{}, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close response body: %w", closeErr)
		}
	}()

	result = conditionalResult{ETag: resp.Header.Get("ETag")}

	switch {
	case resp.StatusCode == http.StatusNotModified:
		result.NotModified = true
		return result, nil
	case resp.StatusCode == http.StatusForbidden:
		return conditionalResult{}, fmt.Errorf("rate limited (HTTP 403)")
	case resp.StatusCode != http.StatusOK:
		return conditionalResult{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return conditionalResult{}, err
	}
	return result, nil
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx != -1 {
		return s[:idx]
	}
	return s
}
