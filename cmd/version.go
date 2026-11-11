package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	// Version is set at release time with -ldflags "-X .../cmd.Version=...".
	Version         = "dev"
	TemplateVersion = "1"
)

// displayVersion is what `dev-box version` and `--version` print.
var displayVersion = Version

// resolvedVersion keeps an injected release version and otherwise falls back to
// what the Go toolchain recorded: the module version for `go install
// module@version`, or the commit for a build made inside a checkout.
func resolvedVersion(injected string, info *debug.BuildInfo) string {
	if injected != "dev" || info == nil {
		return injected
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	revision, modified := "", false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision == "" {
		return injected
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}
	version := injected + "+" + revision
	if modified {
		version += "-dirty"
	}
	return version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI and template versions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return writeCommandOutput(cmd.OutOrStdout(), "dev-box %s\ntemplate schema %s\n", displayVersion, TemplateVersion)
	},
}

func init() {
	info, _ := debug.ReadBuildInfo()
	displayVersion = resolvedVersion(Version, info)
	rootCmd.Version = displayVersion
	rootCmd.AddCommand(versionCmd)
}
