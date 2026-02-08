package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// githubToken is read once at init time from the environment.
var githubToken = os.Getenv("GITHUB_TOKEN")

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
	if githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+githubToken)
	}
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
		if resetStr := resp.Header.Get("X-RateLimit-Reset"); resetStr != "" {
			if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
				wait := time.Until(time.Unix(resetUnix, 0)).Truncate(time.Second)
				if wait > 0 {
					return conditionalResult{}, fmt.Errorf("rate limited (resets in %s)", wait)
				}
			}
		}
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
