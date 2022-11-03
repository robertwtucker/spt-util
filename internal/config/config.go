//
// Copyright (c) 2022 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package config

import "fmt"

// AppName represents the name of the application
const AppName = "spt-util"

var (
	appVersion = "development"
	revision   = "unknown"
)

// VersionInfo represents the application's latest version tag and Git revision
type VersionInfo struct {
	Version  string `mapstructure:"version"`
	Revision string `mapstructure:"revision"`
}

// AppVersion returns the application's latest version and Git revision
func AppVersion() VersionInfo { return VersionInfo{Version: appVersion, Revision: revision} }

// String returns a formatted form of the version and revision
func (v VersionInfo) String() string {
	return fmt.Sprintf("%s-%s", v.Version, v.Revision)
}

// Setting keys
const (
	GlobalReleaseKey     = "global.release"
	GlobalNamespaceKey   = "global.namespace"
	DemoUsernameKey      = "demo.username"
	DemoPasswordKey      = "demo.password"
	DemoInitEnvFileKey   = "demo.init.envFile"
	DemoInitChsFileKey   = "demo.init.chsFile"
	DemoInitWorkflowsKey = "demo.init.workflows"
	DemoStageFilesKey    = "demo.stage.files"
)

// Environment variables
const (
	DemoUsernameEnv = "SCALER_USER"
	DemoPasswordEnv = "SCALER_PASS"
)
