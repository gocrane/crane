package app

import (
	"github.com/kolide/kit/version"
	"github.com/spf13/cobra"
)

func NewCmdVersion() *cobra.Command {
	// flags
	var (
		fFull bool
	)
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print craned version",
		Long: `
Print version information and related build info`,
		Run: func(cmd *cobra.Command, args []string) {
			if fFull {
				version.PrintFull()
				return
			}
			version.Print()
		},
	}

	versionCmd.PersistentFlags().BoolVar(&fFull, "full", false, "print full version information")

	return versionCmd
}
