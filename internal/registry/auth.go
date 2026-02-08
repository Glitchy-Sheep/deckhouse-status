package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/glitchy-sheep/deckhouse-status/internal/kube"
)

func fetchToken(ctx context.Context, host, repo string, creds *kube.RegistryCreds) (string, error) {
	// Hit /v2/ to get WWW-Authenticate parameters (realm, service)
	checkURL := fmt.Sprintf("https://%s/v2/", host)
	req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cannot reach registry: %w", err)
	}
	wwwAuth := resp.Header.Get("Www-Authenticate")
	if err := resp.Body.Close(); err != nil {
		return "", fmt.Errorf("close registry response: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		return "", nil // no auth needed
	}
	if wwwAuth == "" {
		return "", fmt.Errorf("no WWW-Authenticate header from registry")
	}

	params := parseWWWAuthenticate(wwwAuth)
	realm := params["realm"]
	if realm == "" {
		return "", fmt.Errorf("no realm in WWW-Authenticate")
	}

	// Build token request URL
	q := url.Values{}
	if svc := params["service"]; svc != "" {
		q.Set("service", svc)
	}
	q.Set("scope", fmt.Sprintf("repository:%s:pull", repo))

	tokenURL := realm + "?" + q.Encode()

	req, err = http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return "", err
	}
	if creds != nil && creds.Auth != "" {
		req.Header.Set("Authorization", "Basic "+creds.Auth)
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request: HTTP %d", resp.StatusCode)
	}

	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("cannot decode token response: %w", err)
	}

	return tokenResp.Token, nil
}

var wwwAuthParamRe = regexp.MustCompile(`(\w+)="([^"]*)"`)

func parseWWWAuthenticate(header string) map[string]string {
	params := make(map[string]string)
	for _, match := range wwwAuthParamRe.FindAllStringSubmatch(header, -1) {
		params[match[1]] = match[2]
	}
	return params
}
