package cmd

import "runtime/debug"

// version is set at release build time with
// -ldflags "-X github.com/bluescorpian/tendactl/cmd.version=<v>".
var version = ""

// buildVersion is what --version prints: the release version when set,
// otherwise the module version `go install ...@vX` records, otherwise "dev".
func buildVersion() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}
