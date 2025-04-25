package cmd

import (
	"Luminary/pkg"
	"fmt"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [manga-id]",
	Short: "Get detailed information about a manga",
	Long:  `Get comprehensive information about a manga, including all chapters.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mangaID := args[0]

		agent, id := pkg.ParseAgentID(mangaID)
		if agent == "" {
			_ = fmt.Errorf("invalid manga ID format, must be 'agent:id'")
			return
		}

		if apiMode {
			// Output machine-readable JSON for Palaxy
			fmt.Printf(`{"manga":{"id":"%s","title":"Example Manga","chapters":[]}}`, id) // Placeholder
		} else {
			// Interactive mode for CLI users
			fmt.Printf("Getting info for manga ID: %s\n", id)
		}
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
