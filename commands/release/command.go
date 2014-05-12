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

package release

import (
	// stdlib
	"errors"
	"os"

	// trunk
	"github.com/tchap/trunk/log"
	"github.com/tchap/trunk/version"

	// others
	"github.com/tchap/gocli"
)

var ErrActionsFailed = errors.New("some of the requested actions have failed")

var Command = &gocli.Command{
	UsageLine: `
  release [-v] [-remote=REMOTE] [-next=NEXT]
        [-skip_milestone_check] [-skip_build_check]`,
	Short: "perform the release shifting operation",
	Long: `
  NEXT must be a version number in the form of x.y, or "auto", in which case
  the next release number is generated from the previous one by incrementing
  the minor number (i.e. the "y" part).

  This subcommand performs the following actions, which can be divided into
  the current release finishing part and the next release initialization part:

  Finishing of the current release:
    1. Make sure that the workspace and Git index are clean.
    2. Make sure that develop, release and master are up to date.
    3. Make sure that the release CI builds are green.
    4. Make sure that all the assigned GitHub issues are closed.
    5. Reset master to point to the current release.
    6. Commit the new production version string.
    7. Tag master with the appropriate release tag.
    8. Close the relevant GitHub milestone.

  Initialization of the next release:
    1. Reset release to point to develop and commit the new version string.
    2. Increment the minor version number and commit it into develop.
    3. Create a new GitHub milestone for the next release.

  Finalization:
    1. Push the release tag and all the branches (develop, release, master).
	`,
	Action: run,
}

var (
	verbose            bool
	remote             string = "origin"
	next               string = "auto"
	skipMilestoneCheck bool
	skipBuildCheck     bool
)

func init() {
	Command.Flags.BoolVar(&verbose, "v", verbose,
		"print more verbose output")
	Command.Flags.StringVar(&remote, "remote", remote,
		"Git remote to modify")
	Command.Flags.StringVar(&next, "next", next,
		"version number to use for the next release")
	Command.Flags.BoolVar(&skipMilestoneCheck, "skip_milestone_check", skipMilestoneCheck,
		"skip the GitHub milestone check before closing it")
	Command.Flags.BoolVar(&skipBuildCheck, "skip_build_check", skipBuildCheck,
		"skip the Circle CI release build status check")
}

func run(cmd *gocli.Command, args []string) {
	// Parse the arguments.
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	if verbose {
		log.SetV(log.Verbose)
	}

	// Perform the shifting.
	if err := shift(next); err != nil {
		log.Fatalln("\nFatal: ", err)
	}
}

func shift(next string) (err error) {
	// Parse the relevant Git remote to get the GitHub repository name and owner.
	log.V(log.Verbose).Run("Read the GitHub repository name and owner")
	repoOwner, repoName, stderr, err := getGitHubOwnerAndRepository()
	if err != nil {
		log.Print(stderr)
		return
	}

	// Read the current version strings.
	log.V(log.Verbose).Run("Load the current version strings")
	versions, stderr, err := version.LoadVersions()
	if err != nil {
		log.Print(stderr)
		return
	}
	nextVersions, err := versions.Next(next)
	if err != nil {
		return
	}

	// Proceed to the next phase.
	return finishRelease(repoOwner, repoName, versions, nextVersions)
}
