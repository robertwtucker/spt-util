//
// Copyright (c) 2022 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package demo

import (
	"fmt"

	"github.com/robertwtucker/spt-util/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// InitCmd represents the init command
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "initializes a demo instance",
	Long: `
Initializes a demo instance given the specified release and namespace.
    `,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("init called")
		fmt.Println("Release: " + viper.GetString(config.GlobalReleaseKey))
	},
}

func init() {

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
