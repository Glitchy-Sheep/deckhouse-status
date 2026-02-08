package github

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"golang.org/x/sync/errgroup"
)

const apiBase = "https://api.github.com"

func fetchCommitInfo(ctx context.Context, owner, repo, sha string) (commitInfo, error) {
	commitURL := fmt.Sprintf("%s/repos/%s/%s/commits/%s", apiBase, owner, repo, sha)
	var resp commitResponse
	if err := getJSON(ctx, commitURL, &resp); err != nil {
		return commitInfo{}, err
	}
	info := commitInfo{
		Author:  resp.Commit.Author.Name,
		Message: firstLine(resp.Commit.Message),
	}
	if t, err := time.Parse(time.RFC3339, resp.Commit.Author.Date); err == nil {
		info.Date = t
	}
	return info, nil
}

func fetchCheckRun(ctx context.Context, owner, repo, sha, checkName string) (*CheckRunResult, error) {
	checkURL := fmt.Sprintf("%s/repos/%s/%s/commits/%s/check-runs?check_name=%s&per_page=1",
		apiBase, owner, repo, sha, url.QueryEscape(checkName))
	var resp checkRunsResponse
	if err := getJSON(ctx, checkURL, &resp); err != nil {
		return nil, err
	}
	result := &CheckRunResult{}
	if len(resp.CheckRuns) > 0 {
		cr := resp.CheckRuns[0]
		result.Status = cr.Status
		result.Conclusion = cr.Conclusion
		if t, err := time.Parse(time.RFC3339, cr.CompletedAt); err == nil {
			result.CompletedAt = t
		}
	}
	return result, nil
}

// FetchHeadSHA returns the head commit SHA for a pull request (1 API call).
func FetchHeadSHA(ctx context.Context, owner, repo string, prNumber int) (string, error) {
	prURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", apiBase, owner, repo, prNumber)
	var resp pullRequestResponse
	if err := getJSON(ctx, prURL, &resp); err != nil {
		return "", fmt.Errorf("fetch PR #%d head SHA: %w", prNumber, err)
	}
	return resp.Head.SHA, nil
}

// PollCheckRun fetches a single check-run with ETag support for efficient polling.
// When ETag is provided and server returns 304, result.NotModified will be true.
func PollCheckRun(ctx context.Context, req PollCheckRunRequest) (*CheckRunResult, error) {
	checkURL := fmt.Sprintf("%s/repos/%s/%s/commits/%s/check-runs?check_name=%s&per_page=1",
		apiBase, req.Owner, req.Repo, req.SHA, url.QueryEscape(req.CheckName))

	var resp checkRunsResponse
	cond, err := getJSONConditional(ctx, checkURL, req.ETag, &resp)
	if err != nil {
		return nil, err
	}

	result := &CheckRunResult{ETag: cond.ETag, NotModified: cond.NotModified}
	if cond.NotModified {
		return result, nil
	}

	if len(resp.CheckRuns) > 0 {
		cr := resp.CheckRuns[0]
		result.Status = cr.Status
		result.Conclusion = cr.Conclusion
		if t, err := time.Parse(time.RFC3339, cr.CompletedAt); err == nil {
			result.CompletedAt = t
		}
	}

	return result, nil
}

// FetchPRInfo fetches PR info, last commit details, and CI build status.
// Uses 2-3 GitHub API calls (public, no token needed).
// When skipCommitDetails is true, skips the commit details call (2 calls instead of 3).
func FetchPRInfo(ctx context.Context, owner, repo string, prNumber int, buildCheckName string, skipCommitDetails bool) (*PRInfo, error) {
	prURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", apiBase, owner, repo, prNumber)

	var prResp pullRequestResponse
	if err := getJSON(ctx, prURL, &prResp); err != nil {
		return nil, fmt.Errorf("fetch PR #%d: %w", prNumber, err)
	}

	info := &PRInfo{
		Number:         prNumber,
		Title:          prResp.Title,
		URL:            prResp.HTMLURL,
		HeadSHA:        prResp.Head.SHA,
		BuildCheckName: buildCheckName,
	}
	if t, err := time.Parse(time.RFC3339, prResp.UpdatedAt); err == nil {
		info.UpdatedAt = t
	}

	// Fetch commit details and check-runs in parallel.
	var g errgroup.Group

	if !skipCommitDetails {
		g.Go(func() error {
			c, err := fetchCommitInfo(ctx, owner, repo, info.HeadSHA)
			if err != nil {
				return fmt.Errorf("fetch commit %s: %w", info.HeadSHA, err)
			}
			info.CommitAuthor = c.Author
			info.CommitDate = c.Date
			info.CommitMessage = c.Message
			return nil
		})
	}

	g.Go(func() error {
		cr, err := fetchCheckRun(ctx, owner, repo, info.HeadSHA, buildCheckName)
		if err != nil {
			return fmt.Errorf("fetch check-run %q for %s: %w", buildCheckName, info.HeadSHA, err)
		}
		info.BuildStatus = cr.Status
		info.BuildConclusion = cr.Conclusion
		info.BuildCompletedAt = cr.CompletedAt
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return info, nil
}
