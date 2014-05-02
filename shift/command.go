// Copyright (c) 2014 The AUTHORS
//
// This file is part of trunk.
//
// trunk is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// trunk is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with trunk.  If not, see <http://www.gnu.org/licenses/>.

package shift

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"

	"github.com/tchap/gocli"
)

var Command = &gocli.Command{
	UsageLine: "shift [-remote=REMOTE] [-skip_milestones] NEXT_VERSION",
	Short:     "perform the TBD branch shifting operation",
	Long: `
  This subcommand performs the following actions:

    1. Check that the workspace and the index are clean.
    2. Pull develop, release and master.
    3. Close the current GitHub release milestone.
       This operation will fail unless all the assigned issues are closed.
    4. Tag the current release branch.
    5. Reset the master branch to point to the newly created release tag.
    6. Reset the release branch to point to develop (trunk).
    7. If package.json is present in the repository, write the new version
       into the file and commit it into the release branch.
    8. Push the release tag and all the branches (develop, release, master).
    9. Create a new GitHub milestone for the next release.

  The milestones-handling steps are skipped when -skip_milestones is set.
	`,
	Action: run,
}

var (
	remote         string = "origin"
	skipMilestones bool
)

func init() {
	Command.Flags.StringVar(&remote, "remote", remote,
		"Git remote to modify")
	Command.Flags.BoolVar(&skipMilestones, "skip_milestones", skipMilestones,
		"Skip the milestones steps")
}

var (
	ErrDirty = errors.New("the repository is dirty")
)

func run(cmd *gocli.Command, args []string) {
	log.SetFlags(0)

	// Parse the arguments.
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(2)
	}

	next := args[0]
	matched, err := regexp.Match("^[0-9]+([.][0-9]){2}$", []byte(next))
	if err != nil {
		log.Fatal(err)
	}
	if !matched {
		log.Fatal("Invalid version string")
	}

	// Perform the shifting.
	if err := shift(args[0]); err != nil {
		log.Fatal("\nError: ", err)
	}
}

func shift(next string) (err error) {
	// Step 1: Make sure that the workspace and the index are clean.
	log.Print("---> Performing the initial repository checkup")
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return
	}
	if len(out) != 0 {
		return ErrDirty
	}

	// Step 1: Pull the relevant branches.
	log.Printf("---> Fetching %v\n", remote)
	if e := exec.Command("git", "remote", "update", remote).Run(); e != nil {
		return e
	}

	currentRaw, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return
	}
	current := string(bytes.TrimSpace(currentRaw))
	defer func() {
		log.Printf("---> Checking out the original branch (%v)\n", current)
		if e := exec.Command("git", "checkout", current).Run(); e != nil {
			err = e
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		for {
			<-signalCh
			log.Print("Signal received and ignored, wait for the action to finish.")
		}
	}()

	for _, branch := range [...]string{"develop", "release"} {
		log.Printf("---> Merging %v/%v into %v (fast-forward only)", remote, branch, branch)
		if e := exec.Command("git", "checkout", branch).Run(); e != nil {
			return e
		}

		if e := exec.Command("git", "merge", "--ff-only", remote+"/"+branch).Run(); e != nil {
			return e
		}
	}

	return nil
}
