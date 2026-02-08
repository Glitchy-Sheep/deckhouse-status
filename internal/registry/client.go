package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/glitchy-sheep/deckhouse-status/internal/kube"
)

var ErrTagNotFound = errors.New("tag not found")

type Result struct {
	TagExists   bool   // true if registry tag still exists
	Digest      string // digest from registry (when tag exists)
	DigestMatch bool   // true if registry digest matches running digest
	ImageExists bool   // true if image blob still exists by digest (when tag is gone)
	Err         error
}

// Check verifies the image tag in registry and compares digests.
func Check(ctx context.Context, host, repo, tag, runningDigest string, creds *kube.RegistryCreds) *Result {
	r := &Result{}

	if host == "" || repo == "" || tag == "" {
		r.Err = fmt.Errorf("incomplete image reference")
		return r
	}

	// Get Bearer token for this repository
	token, err := fetchToken(ctx, host, repo, creds)
	if err != nil {
		r.Err = fmt.Errorf("registry auth: %w", err)
		return r
	}

	// Check if tag exists and get its digest
	digest, err := fetchManifestDigest(ctx, host, repo, tag, token)
	if err == nil {
		r.TagExists = true
		r.Digest = digest
		r.DigestMatch = digest == runningDigest
		return r
	}

	if !errors.Is(err, ErrTagNotFound) {
		r.Err = err
		return r
	}

	// Tag not found — check if image still exists by its running digest
	r.TagExists = false
	if runningDigest != "" {
		_, err := fetchManifestDigest(ctx, host, repo, runningDigest, token)
		r.ImageExists = err == nil
	}

	return r
}

func fetchManifestDigest(ctx context.Context, host, repo, reference, token string) (string, error) {
	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", host, repo, reference)

	req, err := http.NewRequestWithContext(ctx, "HEAD", manifestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return "", fmt.Errorf("close registry response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", ErrTagNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", fmt.Errorf("no Docker-Content-Digest header")
	}

	return digest, nil
}
