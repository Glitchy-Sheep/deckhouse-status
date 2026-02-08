package display

import (
	"fmt"
	"os"
	"time"

	"github.com/glitchy-sheep/deckhouse-status/internal/github"
	"github.com/glitchy-sheep/deckhouse-status/internal/kube"
	"github.com/glitchy-sheep/deckhouse-status/internal/registry"
)

func (p *Printer) printHeader() {
	now := time.Now().In(p.loc)
	_, offset := now.Zone()
	hours := offset / 3600
	minutes := (offset % 3600) / 60

	var tz string
	if minutes == 0 {
		tz = fmt.Sprintf("UTC%+d", hours)
	} else {
		tz = fmt.Sprintf("UTC%+d:%02d", hours, abs(minutes))
	}

	fmt.Println()
	fmt.Printf("%s%s%s Deckhouse Status%s\n", p.bold, p.white, p.emoji("🔍", ">>"), p.reset)
	fmt.Printf("   %s%s%s (%s)\n", p.dim, now.Format("Mon, 02 Jan 2006 15:04:05"), p.reset, tz)
	fmt.Println()
}

func (p *Printer) printCluster(c *kube.ClusterInfo) {
	p.section(p.emoji("☸️ ", "[K8S]") + " CLUSTER")
	p.row(p.emoji("🏷️", "*"), "Image", p.bold+c.Tag+p.reset)
	p.row(p.emoji("🔄", ">"), "Updated", c.PodCreated.In(p.loc).Format("2006-01-02 15:04"))
	p.row(p.emoji("⏱️", "T"), "Pod age", fmt.Sprintf("%s %s(%s)%s", humanDuration(time.Since(c.PodCreated)), p.dim, c.PodPhase, p.reset))
	p.row(p.emoji("📦", "P"), "Pod", p.dim+c.PodName+p.reset)
	fmt.Println()
}

func (p *Printer) printGitHub(pr *github.PRInfo, prErr error) {
	p.section(p.emoji("🐙", "[GH]") + " GITHUB")

	if prErr != nil {
		p.row(p.emoji("⚠️", "!"), "Error", p.red+prErr.Error()+p.reset)
		fmt.Println()
		return
	}
	if pr == nil {
		p.row(p.emoji("⚠️", "!"), "Error", p.red+"no data"+p.reset)
		fmt.Println()
		return
	}

	p.row(p.emoji("📝", "#"), "PR", fmt.Sprintf("%s#%d%s — %s", p.bold, pr.Number, p.reset, pr.Title))
	p.row(p.emoji("🔗", "~"), "URL", p.dim+pr.URL+p.reset)

	if pr.CommitAuthor != "" {
		dateStr := ""
		if !pr.CommitDate.IsZero() {
			dateStr = fmt.Sprintf(" %s(%s)%s", p.dim, pr.CommitDate.In(p.loc).Format("2006-01-02"), p.reset)
		}
		p.row(p.emoji("👤", "@"), "Last commit", pr.CommitAuthor+dateStr)
	}
	if pr.CommitMessage != "" {
		msg := pr.CommitMessage
		if len(msg) > 70 {
			msg = msg[:67] + "..."
		}
		p.row(p.emoji("💬", ">"), "Message", p.dim+msg+p.reset)
	}
	fmt.Println()
}

func (p *Printer) printStatus(cluster *kube.ClusterInfo, pr *github.PRInfo, reg *registry.Result, edition string) {
	p.section(p.emoji("📊", "[ST]") + " STATUS")

	switch {
	case reg != nil && reg.TagExists && reg.Digest != "":
		if reg.DigestMatch {
			p.statusRow(p.emoji("✅", "[OK]"), p.green, "Up to date", "digest matches registry")
		} else {
			p.statusRow(p.emoji("⚠️", "[!]"), p.yellow, "Outdated", "registry has newer image for tag")
			p.row(p.emoji("🔄", "->"), "Action", p.yellow+"restart pod to update"+p.reset)
		}

	case pr != nil && pr.BuildStatus != "":
		p.printBuildBasedStatus(cluster, pr, edition)

	case pr != nil && pr.BuildStatus == "":
		p.statusRow(p.emoji("⏳", "[..]"), p.cyan, "Waiting for CI", fmt.Sprintf("Build %s not started yet", edition))

	case reg != nil && reg.Err != nil:
		p.statusRow(p.emoji("❓", "[?]"), "", "Cannot determine", fmt.Sprintf("registry: %s", reg.Err))

	default:
		p.statusRow(p.emoji("❓", "[?]"), "", "Cannot determine", "no registry tag, no build info")
	}

	if reg != nil && !p.cfg.NoRegistry {
		p.printRegistryLine(reg)
	}

	fmt.Println()
}

func (p *Printer) printBuildBasedStatus(cluster *kube.ClusterInfo, pr *github.PRInfo, edition string) {
	buildName := "Build " + edition

	switch {
	case pr.BuildStatus == "in_progress" || pr.BuildStatus == "queued":
		p.statusRow(p.emoji("🔄", "[~]"), p.cyan, "Building", fmt.Sprintf("%s is running...", buildName))

	case pr.BuildConclusion == "success" && !pr.BuildCompletedAt.IsZero():
		podTime := cluster.PodCreated
		buildTime := pr.BuildCompletedAt
		podFmt := podTime.In(p.loc).Format("15:04")
		buildFmt := buildTime.In(p.loc).Format("15:04")

		if podTime.After(buildTime) {
			p.statusRow(p.emoji("✅", "[OK]"), p.green, "Up to date", fmt.Sprintf("pod created after %s: %s > %s", buildName, podFmt, buildFmt))
		} else {
			p.statusRow(p.emoji("⚠️", "[!]"), p.yellow, "Outdated", fmt.Sprintf("%s completed after pod: %s > %s", buildName, buildFmt, podFmt))
			p.row(p.emoji("🔄", "->"), "Action", p.yellow+"restart pod to update"+p.reset)
		}

	case pr.BuildConclusion == "failure":
		p.statusRow(p.emoji("❌", "[X]"), p.red, "Build failed", fmt.Sprintf("%s failed on last commit", buildName))

	default:
		p.statusRow(p.emoji("❓", "[?]"), "", "Cannot determine", fmt.Sprintf("%s status: %s", buildName, pr.BuildStatus))
	}
}

func (p *Printer) printRegistryLine(reg *registry.Result) {
	var msg string
	switch {
	case reg.Err != nil:
		msg = fmt.Sprintf("error (%s)", reg.Err)
	case reg.TagExists:
		msg = "tag available"
	case reg.ImageExists:
		msg = "tag removed (image exists by digest)"
	default:
		msg = "tag removed"
	}
	p.row(p.emoji("ℹ️", "i"), "Registry", p.dim+msg+p.reset)
}

// statusLine returns a one-line status string (for short mode).
func (p *Printer) statusLine(cluster *kube.ClusterInfo, pr *github.PRInfo, reg *registry.Result, edition string) string {
	switch {
	case reg != nil && reg.TagExists && reg.Digest != "":
		if reg.DigestMatch {
			return fmt.Sprintf("%s%s Up to date%s", p.green, p.emoji("✅", "[OK]"), p.reset)
		}
		return fmt.Sprintf("%s%s Outdated%s", p.yellow, p.emoji("⚠️", "[!]"), p.reset)

	case pr != nil && pr.BuildStatus != "":
		buildName := "Build " + edition
		switch {
		case pr.BuildStatus == "in_progress" || pr.BuildStatus == "queued":
			return fmt.Sprintf("%s%s Building...%s", p.cyan, p.emoji("🔄", "[~]"), p.reset)
		case pr.BuildConclusion == "success" && !pr.BuildCompletedAt.IsZero():
			if cluster.PodCreated.After(pr.BuildCompletedAt) {
				return fmt.Sprintf("%s%s Up to date%s", p.green, p.emoji("✅", "[OK]"), p.reset)
			}
			return fmt.Sprintf("%s%s Outdated%s %s(new %s)%s", p.yellow, p.emoji("⚠️", "[!]"), p.reset, p.dim, buildName, p.reset)
		case pr.BuildConclusion == "failure":
			return fmt.Sprintf("%s%s Build failed%s", p.red, p.emoji("❌", "[X]"), p.reset)
		}

	case pr != nil && pr.BuildStatus == "":
		return fmt.Sprintf("%s%s Waiting for CI...%s", p.cyan, p.emoji("⏳", "[..]"), p.reset)
	}

	return fmt.Sprintf("%s Unknown%s", p.emoji("❓", "[?]"), p.reset)
}

// PrintWatchHeader prints a brief header for the watch-build command to stderr.
func (p *Printer) PrintWatchHeader(prNumber int, edition, sha string) {
	shortSHA := sha
	if len(shortSHA) > 12 {
		shortSHA = shortSHA[:12]
	}
	fmt.Fprintf(os.Stderr, "%s%sWatching Build %s for PR #%d%s\n", p.bold, p.cyan, edition, prNumber, p.reset)
	fmt.Fprintf(os.Stderr, "%sCommit: %s%s\n\n", p.dim, shortSHA, p.reset)
}
