//
// Copyright (c) 2022 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package demo

import (
	cp "github.com/otiai10/copy"
	"github.com/robertwtucker/spt-util/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// StageCmd represents the stage command
var StageCmd = &cobra.Command{
	Use:   "stage",
	Short: "stages demo resources",
	Long: `
Stages resources for a demo instance
	`,
	Run: func(cmd *cobra.Command, args []string) {
		executeStage()
	},
}

func init() {}

type FilesToCopy struct {
	Source      string `mapstructure:"src"`
	Destination string `mapstructure:"dest"`
}

func executeStage() {
	log.Info("starting demo environment file staging")

	var files []FilesToCopy
	err := viper.UnmarshalKey(config.DemoStageFilesKey, &files)
	if err != nil {
		log.Fatal("error getting config file values")
	}

	log.Infof("file(s) to process: %d", len(files))
	for _, f := range files {
		log.WithFields(log.Fields{
			"src":  f.Source,
			"dest": f.Destination,
		}).Info("copying file")

		err := cp.Copy(f.Source, f.Destination)
		if err != nil {
			log.Fatalf("error copying file: %s", err)
		}
	}

	log.Info("completed staging demo environment files")
}
