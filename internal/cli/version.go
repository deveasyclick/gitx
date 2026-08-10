package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the gitx version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("gitx version %s\n", currentVersion)
			return nil
		},
	}
}
