package main

import (
	"os"

	cc "github.com/ivanpirog/coloredcobra"
	"github.com/spf13/cobra"

	"github.com/glitchy-sheep/deckhouse-status/internal/display"
	"github.com/glitchy-sheep/deckhouse-status/internal/motd"
)

var (
	version = "dev"
	cfg     display.Config
)

var rootCmd = &cobra.Command{
	Use:     "deckhouse-status",
	Short:   "Show Deckhouse deployment status on a dev cluster",
	Version: version,
	Long: `Checks the running Deckhouse pod, compares it with the latest CI build,
and shows whether your PR deployment is up to date.

Data sources: Kubernetes API, GitHub public API, Docker Registry v2.`,
	Run:          runStatus,
	SilenceUsage: true,
}

var installMotdCmd = &cobra.Command{
	Use:   "install-motd",
	Short: "Install login script to show status on every SSH session",
	Long:  `Creates a script in /etc/update-motd.d/ (or /etc/profile.d/ as fallback) that runs deckhouse-status on login.`,
	Run: func(cmd *cobra.Command, args []string) {
		motd.Install()
	},
}

var uninstallMotdCmd = &cobra.Command{
	Use:   "uninstall-motd",
	Short: "Remove the login script",
	Run: func(cmd *cobra.Command, args []string) {
		motd.Uninstall()
	},
}

var editMotdCmd = &cobra.Command{
	Use:   "edit-motd",
	Short: "Open the login script in $EDITOR to customize flags",
	Run: func(cmd *cobra.Command, args []string) {
		motd.Edit()
	},
}

func init() {
	rootCmd.Flags().BoolVarP(&cfg.Short, "short", "s", false, "Compact output (3 lines)")
	rootCmd.Flags().BoolVar(&cfg.NoGitHub, "no-github", false, "Skip GitHub API calls")
	rootCmd.Flags().BoolVar(&cfg.NoRegistry, "no-registry", false, "Skip registry checks")
	rootCmd.Flags().IntVar(&cfg.Timeout, "timeout", 15, "Timeout in seconds")

	// Persistent flags available to all subcommands
	rootCmd.PersistentFlags().StringVar(&cfg.TZ, "tz", "Europe/Moscow", "Timezone: IANA name or numeric offset (+3, -5)")
	rootCmd.PersistentFlags().BoolVar(&cfg.NoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVar(&cfg.NoEmoji, "no-emoji", false, "Disable emojis")

	// watch-build flags
	watchBuildCmd.Flags().IntVar(&watchTimeout, "timeout", 3600, "Timeout in seconds (default 60min)")
	watchBuildCmd.Flags().BoolVar(&watchRestart, "restart", false, "Restart deckhouse deployment on successful build")

	rootCmd.AddCommand(installMotdCmd)
	rootCmd.AddCommand(uninstallMotdCmd)
	rootCmd.AddCommand(editMotdCmd)
	rootCmd.AddCommand(watchBuildCmd)
}

func main() {
	cc.Init(&cc.Config{
		RootCmd:       rootCmd,
		Headings:      cc.HiCyan + cc.Bold + cc.Underline,
		Commands:      cc.HiYellow + cc.Bold,
		Aliases:       cc.Bold + cc.Italic,
		ExecName:      cc.Bold,
		Flags:         cc.Bold,
		FlagsDataType: cc.Italic + cc.HiBlue,
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
