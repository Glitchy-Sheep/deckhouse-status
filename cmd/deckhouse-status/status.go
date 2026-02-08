package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/glitchy-sheep/deckhouse-status/internal/display"
	"github.com/glitchy-sheep/deckhouse-status/internal/github"
	"github.com/glitchy-sheep/deckhouse-status/internal/kube"
	"github.com/glitchy-sheep/deckhouse-status/internal/registry"
)

func runStatus(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	client, err := kube.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cluster, err := client.FetchClusterInfo(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	prNumber, edition := display.ParsePRTag(cluster.Tag)

	var (
		prInfo *github.PRInfo
		prErr  error
		regRes *registry.Result
		wg     sync.WaitGroup
	)

	if prNumber > 0 && !cfg.NoGitHub {
		wg.Add(1)
		go func() {
			defer wg.Done()
			prInfo, prErr = github.FetchPRInfo(ctx, "deckhouse", "deckhouse", prNumber, "Build "+edition, cfg.Short)
		}()
	}

	if !cfg.NoRegistry {
		wg.Add(1)
		go func() {
			defer wg.Done()
			regRes = registry.Check(ctx, cluster.Registry, cluster.Repository, cluster.Tag, cluster.RunningDigest, cluster.RegistryCreds)
		}()
	}

	wg.Wait()

	p := display.NewPrinter(cfg)
	p.Render(display.RenderData{
		Cluster:  cluster,
		PRNumber: prNumber,
		Edition:  edition,
		PR:       prInfo,
		PRErr:    prErr,
		Registry: regRes,
	})
}
