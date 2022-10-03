//
// Copyright (c) 2022 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package cmd

import (
	"github.com/robertwtucker/spt-util/cmd/demo"

	"github.com/spf13/cobra"
)

// demoCmd represents the demo command
var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "demo resource operations",
	Long: `
Performs operations against a set of demo resources
	`,
}

func init() {
	demoCmd.AddCommand(demo.InitCmd)
	demoCmd.AddCommand(demo.StageCmd)
	rootCmd.AddCommand(demoCmd)
}
