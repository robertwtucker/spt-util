//
// Copyright (c) 2023 Quadient Group AG
//
// This file is subject to the terms and conditions defined in the
// 'LICENSE' file found in the root of this source code package.
//

package version

import "fmt"

// Version variables used for build process.
var (
	version  = "development"
	revision = "unset"
)

// GetVersion returns the application's version.
func GetVersion() string {
	return fmt.Sprintf("%s-%s", version, revision)
}

// GetRevision returns the application's git revision.
func GetRevision() string {
	return revision
}

// GetSemVersion() returns the application's version (SemVer format).
func GetSemVersion() string {
	return fmt.Sprintf("v%s", version)
}
