//
// Copyright (c) 2022 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package constants

// AppName represents the name of the application.
const AppName = "spt-util"

// Setting keys.
const (
	GlobalReleaseKey     = "global.release"
	GlobalNamespaceKey   = "global.namespace"
	DemoUsernameKey      = "demo.username"
	DemoPasswordKey      = "demo.password"
	DemoServerKey        = "demo.server"
	DemoInitEnvFileKey   = "demo.init.envFile"
	DemoInitChsFileKey   = "demo.init.chsFile"
	DemoInitWorkflowsKey = "demo.init.workflows"
	DemoStageFilesKey    = "demo.stage.files"
)

// Environment variables.
const (
	DemoUsernameEnv = "SCALER_USER"
	DemoPasswordEnv = "SCALER_PASS"
	DemoServerEnv   = "SCALER_URL"
)
