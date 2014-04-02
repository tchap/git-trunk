// Copyright (c) 2014 The AUTHORS
//
// This file is part of git-trunk.
//
// git-trunk is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// git-trunk is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with git-trunk.  If not, see <http://www.gnu.org/licenses/>.

package stage

import (
	"github.com/tchap/gocli"
	"os/exec"
	"strings"
)

var Command = &gocli.Command{
	UsageLine: "update",
	Short:     "update Git remotes",
	Long: `
  This subcommand performs the following actions:

    1. remember the current branch as BRANCH
    2. git remote update upstream
    3. git checkout develop
    4. git merge --ff-only upstream/develop
    5. git checkout $BRANCH
	`,
	Action: run,
}

func run(command *gocli.Command, args []string) {
	// Step 0: Make sure that there were no arguments specified.
	if len(args) != 0 {
		command.Usage()
		os.Exit(2)
	}

	// Step 1
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	currentBranch := strings.TrimSpace(executil.OutputOrFatal(cmd))

	// Step 2
	cmd = exec.Command("git", "remote", "update", config.UpstreamRemote)
	executil.RunOrFatal(cmd)

	// Step 3
	cmd = exec.Command("git", "checkout", config.DevelopBranch)
	executil.RunOrFatal(cmd)

	// Step 4
	remoteBranch = config.UpstreamRemote + "/" + config.DevelopBranch
	cmd = exec.Command("git", "merge", "--ff-only", remoteBranch)
	executil.RunOrFatal(cmd)

	// Step 5
	cmd = exec.Command("git", "checkout", currentBranch)
	executil.RunOrFatal(cmd)

	fmt.Println("\nSuccess")
}
