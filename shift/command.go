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
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"

	"github.com/tchap/gocli"
)

const (
	TrunkBranch   = "develop"
	ReleaseBranch = "release"
)

var (
	ErrDirtyRepository = errors.New("the repository is dirty")
)

var Command = &gocli.Command{
	UsageLine: "shift [-remote=REMOTE] [-skip_milestones] NEXT",
	Short:     "perform the branch shifting operation",
	Long: `
  This subcommand performs the following actions:

    1. Make sure that the workspace and the index are clean.
    2. Reset develop and release to point to their remote counterparts.
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

  NEXT must be a version number in the form of x.y.z, or "auto", in which case
  the next release number is generated from the previous one by incrementing
  the release number (the "z" part).
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
	log.Print("---> Performing the initial repository check")
	output, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return
	}
	if len(output) != 0 {
		return ErrDirtyRepository
	}

	// Step 2: Reset develop and release to point to their remote counterparts.
	log.Printf("---> Fetching %v\n", remote)
	output, err = exec.Command("git", "fetch", remote).CombinedOutput()
	if err != nil {
		log.Print(string(output))
		return
	}

	output, err = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return
	}
	current := string(bytes.TrimSpace(output))
	defer func() {
		log.Printf("---> Checking out the original branch (%v)\n", current)
		output, ex := exec.Command("git", "checkout", current).CombinedOutput()
		if ex != nil {
			log.Print(output)
			if err == nil {
				err = ex
			}
		}
	}()

	// Start blocking os.Interrupt signal, the following actions are fast to
	// perform and there is no reason to try to interrupt them really.
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	closeCh := make(chan struct{})
	defer func() {
		close(closeCh)
	}()
	go func() {
		for {
			select {
			case <-signalCh:
				log.Print("Signal received but ignored. Wait for the action to finish.")
			case <-closeCh:
				return
			}
		}
	}()

	// Make sure that develop and release are synchronized with their remote counterparts.
	trunkRemote := remote + "/" + TrunkBranch
	log.Printf("---> Checking whether %v and %v are synchronized\n", TrunkBranch, trunkRemote)
	err = checkSynchronized(TrunkBranch, trunkRemote)
	if err != nil {
		return
	}

	releaseRemote := remote + "/" + ReleaseBranch
	log.Printf("---> Checking whether %v and %v are synchronized\n", ReleaseBranch, releaseRemote)
	err = checkSynchronized(ReleaseBranch, releaseRemote)
	if err != nil {
		return
	}

	// Rollback develop and release in case something goes wrong.
	output, err = exec.Command("git", "rev-parse", TrunkBranch).Output()
	if err != nil {
		log.Print(string(output))
		return
	}
	trunkSHA := string(bytes.TrimSpace(output))
	defer func() {
		if err != nil {
			reset(TrunkBranch, trunkSHA)
		}
	}()

	output, err = exec.Command("git", "rev-parse", ReleaseBranch).Output()
	if err != nil {
		log.Print(string(output))
		return
	}
	releaseSHA := string(bytes.TrimSpace(output))
	defer func() {
		if err != nil {
			reset(ReleaseBranch, releaseSHA)
		}
	}()

	//    3. Close the current GitHub release milestone.
	//       This operation will fail unless all the assigned issues are closed.
	//    4. Tag the current release branch.
	//    5. Reset the master branch to point to the newly created release tag.
	//    6. Reset the release branch to point to develop (trunk).
	//    7. If package.json is present in the repository, write the new version
	//       into the file and commit it into the release branch.
	//    8. Push the release tag and all the branches (develop, release, master).
	//    9. Create a new GitHub milestone for the next release.

	return nil
}

func reset(branch, hexsha string) {
	log.Printf("---> Resetting %v to the original position\n", branch)

	output, err := exec.Command("git", "checkout", branch).CombinedOutput()
	if err != nil {
		log.Print(string(output))
		return
	}

	output, err = exec.Command("git", "reset", "--hard", hexsha).CombinedOutput()
	if err != nil {
		log.Print(string(output))
		return
	}
}

func checkSynchronized(b1, b2 string) error {
	output, err := exec.Command("git", "diff", "--name-only", b1, b2).Output()
	if err != nil {
		return err
	}
	if len(output) != 0 {
		return fmt.Errorf("refs %v and %v differ", b1, b2)
	}
	return nil
}
