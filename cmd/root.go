//
// Copyright (c) 2022 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/robertwtucker/spt-util/pkg/constants"
	"github.com/robertwtucker/spt-util/pkg/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmdArgs struct {
	ConfigFile string
	LogFormat  string
	LogDebug   bool
	Release    string
	Namespace  string
}

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   constants.AppName,
	Short: "The SPT utility application",
	Long: `
The SPT utility application is used to execute the various scripts necessary to setup
and maintain SPT demo environments.
	`,
	Example: `
# initialize base content for a demo environment with debug logging enabled
spt-util demo init -d

# stage files in a demo environment using a custom configuration file
spt-util demo stage -c <path-to-config.yaml>

# display application version information
spt-util --version
	`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		initLog()
		logrus.WithField("version", version.GetVersion()).Info("initialized")
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

//nolint:gochecknoinits // required for proper cobra initialization
func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&rootCmdArgs.ConfigFile, "config", "c",
		"", "specify the config file (default is ./config/"+constants.AppName+".yaml)")
	rootCmd.PersistentFlags().StringVar(&rootCmdArgs.LogFormat, "log-format",
		"text", "set the logging format [text|json]")
	rootCmd.PersistentFlags().BoolVarP(&rootCmdArgs.LogDebug, "verbose", "d",
		false, "set verbose logging")
	rootCmd.PersistentFlags().StringVarP(&rootCmdArgs.Release, "release", "r",
		"inspire", "set the inspire release name used")
	rootCmd.PersistentFlags().StringVarP(&rootCmdArgs.Namespace, "namespace", "n",
		"default", "set the cluster namespace to target")

	rootCmd.Version = version.GetVersion()             // Enable the version option
	rootCmd.CompletionOptions.DisableDefaultCmd = true // Hide the completion options
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if rootCmdArgs.ConfigFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(rootCmdArgs.ConfigFile)
	} else {
		viper.AddConfigPath("./config")
		viper.SetConfigType("yaml")
		viper.SetConfigName(constants.AppName)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_, _ = fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// Override config file settings with command line arguments, if present.
	if rootCmdArgs.Release != "" {
		viper.Set(constants.GlobalReleaseKey, rootCmdArgs.Release)
	}
	if rootCmdArgs.Namespace != "" {
		viper.Set(constants.GlobalNamespaceKey, rootCmdArgs.Namespace)
	}
}

func initLog() {
	if strings.ToLower(rootCmdArgs.LogFormat) == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
	if rootCmdArgs.LogDebug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	log.SetOutput(logrus.New().Writer())
	log.SetFlags(0)
}
