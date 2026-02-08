package github

import "time"

// GitHub API response types.

type pullRequestResponse struct {
	Title     string `json:"title"`
	HTMLURL   string `json:"html_url"`
	UpdatedAt string `json:"updated_at"`
	Head      struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

type commitResponse struct {
	Commit struct {
		Author struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
}

type checkRunEntry struct {
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion"`
	CompletedAt string `json:"completed_at"`
}

type checkRunsResponse struct {
	CheckRuns []checkRunEntry `json:"check_runs"`
}

// conditionalResult holds the result of a conditional HTTP GET request.
type conditionalResult struct {
	// ETag is the ETag header value from the response.
	ETag string
	// NotModified is true when the server returned 304 Not Modified.
	NotModified bool
}

// Domain types.

// PRInfo contains PR metadata, last commit details, and CI build status.
type PRInfo struct {
	Number    int
	Title     string
	URL       string
	HeadSHA   string
	UpdatedAt time.Time

	CommitAuthor  string
	CommitDate    time.Time
	CommitMessage string // first line only

	BuildCheckName   string // e.g., "Build FE"
	BuildStatus      string // "completed", "in_progress", "queued", ""
	BuildConclusion  string // "success", "failure", ""
	BuildCompletedAt time.Time
}

// PollCheckRunRequest contains parameters for polling a check-run.
type PollCheckRunRequest struct {
	Owner     string
	Repo      string
	SHA       string // head commit SHA of the PR; check runs are attached to commits
	CheckName string // GitHub Actions check run name to filter by (e.g. "Build FE", "Build EE")
	ETag      string // from previous poll; empty for first request
}

// CheckRunResult represents the result of a single check-run poll.
type CheckRunResult struct {
	Status      string    // "queued", "in_progress", "completed"
	Conclusion  string    // "success", "failure", "cancelled", "timed_out"
	CompletedAt time.Time
	NotModified bool   // true when server returned 304
	ETag        string // pass to next PollCheckRunRequest
}

type commitInfo struct {
	Author  string
	Date    time.Time
	Message string
}
