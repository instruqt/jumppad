package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/instruqt/jumppad/pkg/utils"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:                   "uninstall",
	Short:                 "Uninstall jumppad",
	Long:                  `Uninstall jumppad`,
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// remove the config
		fmt.Println("Removing Shipyard configuration from", utils.JumppadHome())
		err := os.RemoveAll(utils.JumppadHome())
		if err != nil {
			fmt.Println("Error: Unable to remove jumppad configuration", err)
			os.Exit(1)
		}

		// remove the binary
		ep, _ := os.Executable()
		cf, err := filepath.Abs(ep)
		if err != nil {
			fmt.Println("Error: Unable to remove jumppad application", err)
			os.Exit(1)
		}
		fmt.Println("Removing jumppad application from", cf)
		err = os.Remove(cf)
		if err != nil {
			fmt.Println("Error: Unable to remove jumppad application", err)
			os.Exit(1)
		}

		fmt.Println("")
		fmt.Println("jumppad successfully uninstalled")
	},
}
