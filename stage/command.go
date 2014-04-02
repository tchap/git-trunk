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
	"log"
	"os"
	"os/signal"
	"syscall"
	"github.com/tchap/gocli"
)

var (
	issue uint
	story uint
)

var Command = &gocli.Command{
	UsageLine: "stage [-norebase] [-issue=ISSUE] [-story=STORY]",
	Short:     "create a pull request for the current feature branch",
	Long: `
  This subcommand performs the following actions:

    1. Rebase the current branch onto develop.
    2. Push the feature branch into origin, which is a person fork.
    3. Create a pull request on GitHub, optionally referencing ISSUE and STORY,
       where ISSUE is a GitHub issue ID, STORY is a Pivotal Tracker story ID.
	`,
	Action: run,
}

func init() {
	Command.Flags.UintVar(&issue, "issue", issue, "GitHub issue to reference")
	Command.Flags.UintVar(&story, "story", story, "Pivotal Tracker story to reference")
}

func run(cmd *gocli.Command, args []string) {
	// Set up logging.
	log.SetFlags(0)

	// Make sure there were no arguments specified.
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	// STEP 1: Rebase the current branch onto develop.
	cmd := exec.Command("git", "rebase", )
}
