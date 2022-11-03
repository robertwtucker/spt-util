//
// Copyright (c) 2022 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package cmd

import (
	"github.com/robertwtucker/spt-util/cmd/demo"
	"github.com/robertwtucker/spt-util/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// demoCmd represents the demo command
var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Operations with demo resources",
	Long: `
Performs operations against a set of demo resources
	`,
	Example: `
# initialize base content for a demo environment with debug logging enabled
spt-util demo init -d

# stage files in a demo environment using a custom configuration file
spt-util demo stage -c <path-to-config.yaml>
	`,
}

func init() {
	// Get Scaler params from environment, not command line
	_ = viper.BindEnv(config.DemoUsernameKey, config.DemoUsernameEnv)
	_ = viper.BindEnv(config.DemoPasswordKey, config.DemoPasswordEnv)
	_ = viper.BindEnv(config.DemoServerKey, config.DemoServerEnv)

	demoCmd.AddCommand(demo.InitCmd)
	demoCmd.AddCommand(demo.StageCmd)
	rootCmd.AddCommand(demoCmd)
}
