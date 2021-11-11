package version

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var (
	gitVersion   = "crane-%s"
	gitCommit    = "$Format:%H$" // sha1 from git, output of $(git rev-parse HEAD)
	gitTreeState = ""            // state of git tree, either "clean" or "dirty"
	gitTag       = ""
	buildDate    = "1970-01-01T00:00:00Z" // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)

type Version struct {
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
	GitTreeState string `json:"gitTreeState"`
}

func GetVersionInfo() string {
	ver := Version{
		GitVersion:   fmt.Sprintf(gitVersion, gitTag),
		GitCommit:    gitCommit,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		GitTreeState: gitTreeState,
	}
	res, _ := json.Marshal(ver)
	return string(res)
}
