package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/glitchy-sheep/deckhouse-status/internal/display"
	"github.com/glitchy-sheep/deckhouse-status/internal/github"
	"github.com/glitchy-sheep/deckhouse-status/internal/kube"
)

var (
	watchTimeout int
	watchRestart bool
)

var watchBuildCmd = &cobra.Command{
	Use:   "watch-build",
	Short: "Watch CI build until completion, then exit with appropriate code",
	Long: `Polls the GitHub check-runs API for the current PR's CI build,
showing a live spinner with elapsed time. Exits with code 0 on success,
1 on build failure, 2 on error/timeout.

Uses ETag conditional requests to minimize GitHub API rate limit usage
(304 Not Modified responses are free).`,
	Run: runWatchBuild,
}

type watchTarget struct {
	client    *kube.Client
	prNumber  int
	edition   string
	sha       string
	checkName string
}

func resolveWatchTarget(ctx context.Context) (*watchTarget, error) {
	client, err := kube.NewClient()
	if err != nil {
		return nil, err
	}

	kubeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	cluster, err := client.FetchClusterInfo(kubeCtx)
	cancel()
	if err != nil {
		return nil, err
	}

	prNumber, edition := display.ParsePRTag(cluster.Tag)
	if prNumber == 0 {
		return nil, fmt.Errorf("image tag %q is not a PR tag", cluster.Tag)
	}

	sha, err := github.FetchHeadSHA(ctx, "deckhouse", "deckhouse", prNumber)
	if err != nil {
		return nil, err
	}

	return &watchTarget{
		client:    client,
		prNumber:  prNumber,
		edition:   edition,
		sha:       sha,
		checkName: "Build " + edition,
	}, nil
}

func runWatchBuild(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(watchTimeout)*time.Second)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	target, err := resolveWatchTarget(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	p := display.NewPrinter(cfg)
	p.PrintWatchHeader(target.prNumber, target.edition, target.sha)

	spinner := display.NewSpinner(cfg.NoColor, cfg.NoEmoji)

	pollReq := github.PollCheckRunRequest{
		Owner:     "deckhouse",
		Repo:      "deckhouse",
		SHA:       target.sha,
		CheckName: target.checkName,
	}

	// First poll immediately
	lastResult, err := pollCheckRun(ctx, pollReq, spinner)
	if err != nil {
		spinner.Failure(fmt.Sprintf("Error: %v", err))
		os.Exit(2)
	}
	pollReq.ETag = lastResult.ETag
	if code, done := checkBuildDone(lastResult, target.checkName, spinner); done {
		if code == 0 && watchRestart {
			doRestart(ctx, target.client, spinner)
		}
		os.Exit(code)
	}

	const pollInterval = 10 * time.Second
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			spinner.ClearLine()
			if ctx.Err() == context.DeadlineExceeded {
				spinner.Failure("Timeout waiting for build to complete")
			} else {
				fmt.Fprintf(os.Stderr, "\nInterrupted.\n")
			}
			os.Exit(2)

		case <-ticker.C:
			lastResult, err = pollCheckRun(ctx, pollReq, spinner)
			if err != nil {
				spinner.Tick(fmt.Sprintf("%s: error (%v), retrying...", target.checkName, err))
				continue
			}
			pollReq.ETag = lastResult.ETag
			if code, done := checkBuildDone(lastResult, target.checkName, spinner); done {
				if code == 0 && watchRestart {
					doRestart(ctx, target.client, spinner)
				}
				os.Exit(code)
			}
		}
	}
}

func pollCheckRun(ctx context.Context, req github.PollCheckRunRequest, spinner *display.Spinner) (*github.CheckRunResult, error) {
	result, err := github.PollCheckRun(ctx, req)
	if err != nil {
		return nil, err
	}

	if result.NotModified {
		spinner.Tick(fmt.Sprintf("%s: no change", req.CheckName))
		return result, nil
	}

	switch result.Status {
	case "queued":
		spinner.Tick(fmt.Sprintf("%s: queued", req.CheckName))
	case "in_progress":
		spinner.Tick(fmt.Sprintf("%s: in progress", req.CheckName))
	case "completed":
		// Let checkBuildDone handle the final output
	default:
		if result.Status == "" {
			spinner.Tick(fmt.Sprintf("%s: waiting for check to appear...", req.CheckName))
		} else {
			spinner.Tick(fmt.Sprintf("%s: %s", req.CheckName, result.Status))
		}
	}

	return result, nil
}

func checkBuildDone(status *github.CheckRunResult, checkName string, spinner *display.Spinner) (exitCode int, done bool) {
	if status == nil || status.Status != "completed" {
		return 0, false
	}

	switch status.Conclusion {
	case "success":
		spinner.Success(fmt.Sprintf("%s completed successfully", checkName))
		return 0, true
	case "failure":
		spinner.Failure(fmt.Sprintf("%s failed", checkName))
		return 1, true
	case "cancelled":
		spinner.Failure(fmt.Sprintf("%s was cancelled", checkName))
		return 1, true
	case "timed_out":
		spinner.Failure(fmt.Sprintf("%s timed out", checkName))
		return 1, true
	default:
		spinner.Failure(fmt.Sprintf("%s completed with: %s", checkName, status.Conclusion))
		return 1, true
	}
}

func doRestart(ctx context.Context, client *kube.Client, spinner *display.Spinner) {
	fmt.Fprintf(os.Stderr, "Restarting deckhouse deployment...\n")

	restartCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := client.RestartDeployment(restartCtx); err != nil {
		spinner.Failure(fmt.Sprintf("Restart failed: %v", err))
	} else {
		spinner.Success("Deployment restarted")
	}
}
