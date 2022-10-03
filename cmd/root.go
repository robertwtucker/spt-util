//
// Copyright (c) 2022 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/robertwtucker/spt-util/internal/config"
	"log"
	"os"
	"strings"

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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   config.AppName,
	Short: "the SPT utility application",
	Long: `
The SPT utility application.

This application is used to execute the various scripts necessary to setup
and maintain SPT demo environments.
	`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := initLog(); err != nil {
			return errors.Wrapf(err, "failed to initialize logging")
		}
		logrus.WithField("version", config.AppVersion().String()).Info("initialized")
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

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&rootCmdArgs.ConfigFile, "config", "c",
		"", "specify the config file (default is ./config/"+config.AppName+".yaml)")
	rootCmd.PersistentFlags().StringVar(&rootCmdArgs.LogFormat, "log-format",
		"text", "set the logging format [text|json]")
	rootCmd.PersistentFlags().BoolVarP(&rootCmdArgs.LogDebug, "verbose", "v",
		false, "set verbose logging")
	rootCmd.PersistentFlags().StringVarP(&rootCmdArgs.Release, "release", "r",
		"inspire", "set the inspire release name used")
	rootCmd.PersistentFlags().StringVarP(&rootCmdArgs.Namespace, "namespace", "n",
		"default", "set the cluster namespace to target")

	rootCmd.Version = config.AppVersion().String()     // Enable the version option
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
		viper.SetConfigName(config.AppName)
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		_, _ = fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// Override config file settings with command line arguments, if present
	if rootCmdArgs.Release != "" {
		viper.Set(config.GlobalReleaseKey, rootCmdArgs.Release)
	}
	if rootCmdArgs.Namespace != "" {
		viper.Set(config.GlobalNamespaceKey, rootCmdArgs.Namespace)
	}
}

func initLog() error {
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

	return nil
}
