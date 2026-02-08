package display

import (
	"fmt"
	"strings"
	"time"

	"github.com/glitchy-sheep/deckhouse-status/internal/github"
	"github.com/glitchy-sheep/deckhouse-status/internal/kube"
	"github.com/glitchy-sheep/deckhouse-status/internal/registry"
)

// Config controls what to display and how.
type Config struct {
	Short      bool
	NoGitHub   bool
	NoRegistry bool
	NoColor    bool
	NoEmoji    bool
	Timeout    int
	TZ         string
}

// RenderData holds all collected data for rendering.
type RenderData struct {
	Cluster  *kube.ClusterInfo
	PRNumber int
	Edition  string
	PR       *github.PRInfo
	PRErr    error
	Registry *registry.Result
}

// Printer handles formatted output with configurable colors and emojis.
type Printer struct {
	cfg Config
	loc *time.Location
	// ANSI codes (empty when NoColor)
	reset, bold, dim          string
	red, green, yellow, cyan  string
	white                     string
}

// NewPrinter creates a Printer with the given config.
func NewPrinter(cfg Config) *Printer {
	loc := parseTZ(cfg.TZ)
	p := &Printer{cfg: cfg, loc: loc}
	if !cfg.NoColor {
		p.reset = "\033[0m"
		p.bold = "\033[1m"
		p.dim = "\033[2m"
		p.red = "\033[31m"
		p.green = "\033[32m"
		p.yellow = "\033[33m"
		p.cyan = "\033[36m"
		p.white = "\033[37m"
	}
	return p
}

// Render prints the full or short status output.
func (p *Printer) Render(d RenderData) {
	if p.cfg.Short {
		p.renderShort(d)
		return
	}
	p.renderFull(d)
}

func (p *Printer) renderFull(d RenderData) {
	p.printHeader()
	p.printCluster(d.Cluster)
	if d.PRNumber > 0 && !p.cfg.NoGitHub {
		p.printGitHub(d.PR, d.PRErr)
	}
	p.printStatus(d.Cluster, d.PR, d.Registry, d.Edition)
}

func (p *Printer) renderShort(d RenderData) {
	status := p.statusLine(d.Cluster, d.PR, d.Registry, d.Edition)

	// Line 1: image tag + pod age + status
	age := humanDuration(time.Since(d.Cluster.PodCreated))
	fmt.Printf("%s %s%s%s %s·%s %s %s·%s %s\n",
		p.emoji("🔍", "*"),
		p.bold, d.Cluster.Tag, p.reset,
		p.dim, p.reset,
		age,
		p.dim, p.reset,
		status,
	)

	// Line 2: PR info (if available)
	if d.PR != nil {
		fmt.Printf("   %s #%d — %s\n", p.emoji("📝", "PR"), d.PR.Number, d.PR.Title)
	}
}

// --- Output helpers ---

func (p *Printer) emoji(emojiStr, fallback string) string {
	if p.cfg.NoEmoji {
		return fallback
	}
	return emojiStr
}

func (p *Printer) section(title string) {
	fmt.Printf("%s━━━ %s ━━━%s\n", p.cyan, title, p.reset)
}

const labelWidth = 19

func (p *Printer) row(icon, label, value string) {
	padding := labelWidth - len(label) - 1
	if padding < 1 {
		padding = 1
	}
	fmt.Printf("%s %s:%s %s\n", icon, label, strings.Repeat(" ", padding), value)
}

func (p *Printer) statusRow(icon, color, title, reason string) {
	value := fmt.Sprintf("%s%s%s  %s(%s)%s", color, title, p.reset, p.dim, reason, p.reset)
	p.row(icon, "Status", value)
}
