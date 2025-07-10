package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type JSBuildResult struct {
	Success    bool
	BuildLog   string
	RunCommand []string
	Error      error
}

// Detects and builds a JS app (Node.js, Next.js, etc.) in the given repoPath.
// Tries npm first, then pnpm if npm fails. Returns build logs and the command to run the app.
func AutoBuildJSApp(ctx context.Context, repoPath string, env map[string]string) JSBuildResult {
	packageJsonPath := filepath.Join(repoPath, "package.json")
	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		return JSBuildResult{Success: false, BuildLog: "No package.json found", Error: fmt.Errorf("no package.json found")}
	}

	// Read package.json
	pkg, err := ioutil.ReadFile(packageJsonPath)
	if err != nil {
		return JSBuildResult{Success: false, BuildLog: "Failed to read package.json", Error: err}
	}
	var pkgJson map[string]interface{}
	if err := json.Unmarshal(pkg, &pkgJson); err != nil {
		return JSBuildResult{Success: false, BuildLog: "Invalid package.json", Error: err}
	}

	// Determine build and start scripts
	buildScript := "build"
	startScript := "start"
	if scripts, ok := pkgJson["scripts"].(map[string]interface{}); ok {
		if _, ok := scripts["build"]; !ok {
			buildScript = ""
		}
		if _, ok := scripts["start"]; !ok {
			startScript = "run"
		}
	}

	// Try npm install/build
	buildLog := ""
	cmds := [][]string{
		{"npm", "install"},
	}
	if buildScript != "" {
		cmds = append(cmds, []string{"npm", "run", buildScript})
	}
	for _, cmdArgs := range cmds {
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		cmd.Dir = repoPath
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
		out, err := cmd.CombinedOutput()
		buildLog += string(out)
		if err != nil {
			// Try pnpm if npm fails
			buildLog += "\n[npm failed, trying pnpm]"
			cmds2 := [][]string{
				{"pnpm", "install"},
			}
			if buildScript != "" {
				cmds2 = append(cmds2, []string{"pnpm", "run", buildScript})
			}
			for _, cmdArgs2 := range cmds2 {
				cmd2 := exec.CommandContext(ctx, cmdArgs2[0], cmdArgs2[1:]...)
				cmd2.Dir = repoPath
				cmd2.Env = os.Environ()
				for k, v := range env {
					cmd2.Env = append(cmd2.Env, fmt.Sprintf("%s=%s", k, v))
				}
				out2, err2 := cmd2.CombinedOutput()
				buildLog += string(out2)
				if err2 != nil {
					return JSBuildResult{Success: false, BuildLog: buildLog, Error: err2}
				}
			}
			break
		}
	}

	// Prepare run command
	runCmd := []string{"npm", "run", startScript}
	if _, err := exec.LookPath("npm"); err != nil {
		if _, err2 := exec.LookPath("pnpm"); err2 == nil {
			runCmd = []string{"pnpm", "run", startScript}
		}
	}

	return JSBuildResult{Success: true, BuildLog: buildLog, RunCommand: runCmd, Error: nil}
}