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
	"log"
	"os"
	"regexp"

	"github.com/libgit2/git2go"
	"github.com/tchap/gocli"
)

var Command = &gocli.Command{
	UsageLine: "shift [-remote=REMOTE] [-skip_milestones] NEXT_VERSION",
	Short:     "perform the TBD branch shifting operation",
	Long: `
  This subcommand performs the following actions:

    1. Pull develop, release and master.
    2. Close the current GitHub release milestone.
       This operation will fail unless all the assigned issues are closed.
    3. Tag the current release branch.
    4. Reset the master branch to point to the newly created release tag.
    5. Reset the release branch to point to develop (trunk).
    6. If package.json is present in the repository, write the new version
       into the file and commit it into the release branch.
    7. Push the release tag and all the branches (develop, release, master).
    8. Create a new GitHub milestone for the next release.

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
		log.Fatal(err)
	}
}

func shift(next string) error {
	// Step 1: Pull the relevant branches.
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	repo, err := git.OpenRepository(wd)
	if err != nil {
		return err
	}

	origin, err := repo.LoadRemote(remote)
	if err != nil {
		return err
	}

	setCallbacks(origin)

	signature, err := getUserSignature(repo)
	if err != nil {
		return err
	}

	log.Printf("---> Fetching %v\n", remote)
	if err := origin.Fetch(signature, "fetch "+remote); err != nil {
		return err
	}

	mergeOpts, err := git.DefaultMergeOptions()
	if err != nil {
		return err
	}

	for _, branch := range [...]string{"develop", "release"} {
		log.Printf("---> Merging %v/%v into %v (fast-forward only)", remote, branch, branch)
		b, err := checkout(repo, branch)
		if err != nil {
			return err
		}

		head, err := repo.MergeHeadFromRef(b.Reference)
		if err != nil {
			return err
		}

		//if err := repo.Merge([]*git.MergeHead{head}, git.DefaultMergeOptions(), )
	}

	return nil
}
