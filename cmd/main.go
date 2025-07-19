package main

import (
	"runtime/debug"
	"time"

	"github.com/mheers/godtemplate/models"
	"github.com/sirupsen/logrus"
)

// build flags
var (
	VERSION   string
	BuildTime string
	GitTag    string
	GitBranch string
)

func main() {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		panic("could not read build info")
	}

	runtimeInfo := &models.RuntimeInfo{}

	for _, kv := range buildInfo.Settings {
		switch kv.Key {
		case "vcs.revision":
			runtimeInfo.CommitHash = kv.Value
		case "vcs.time":
			runtimeInfo.CommitTime, _ = time.Parse(time.RFC3339, kv.Value)
		case "vcs.modified":
			runtimeInfo.DirtyBuild = kv.Value == "true"
		}
	}

	runtimeInfo.Version = VERSION
	runtimeInfo.BuildTime = BuildTime
	runtimeInfo.GoVersion = buildInfo.GoVersion
	runtimeInfo.GitTag = GitTag
	runtimeInfo.GitBranch = GitBranch

	RuntimeInfo = runtimeInfo

	// execeute the command
	err := Execute()
	if err != nil {
		logrus.Fatalf("Execute failed: %+v", err)
	}
}
