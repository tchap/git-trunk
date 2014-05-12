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
	"bytes"

	// trunk
	"github.com/tchap/trunk/config"
	"github.com/tchap/trunk/git"
	"github.com/tchap/trunk/log"
	"github.com/tchap/trunk/version"
)

func startRelease(repoName, repoOwner string, versions, nextVersions *version.Versions) (err error) {
	log.Printf("\n---> Starting release %v\n", nextVersions.Production)

	// Step 1: Reset release to point to develop and commit the new version string.
	log.V(log.Verbose).Run("Reset release to point to develop")
	currentRelease, stderr, err := git.Hexsha(config.Local.Branches.Release)
	if err != nil {
		return failure(stderr, err)
	}

	stderr, err = git.ResetHard(config.Local.Branches.Release, config.Local.Branches.Trunk)
	if err != nil {
		log.Print(stderr)
		log.Println("Error: ", err)
		return
	}
	defer func() {
		if err != nil {
			log.V(log.Verbose).Run("Roll back the release branch")
			stderr, ex := git.ResetHard(config.Local.Branches.Release, currentRelease)
			if ex != nil {
				failure(stderr, ex)
			}
		}
	}()

	log.V(log.Verbose).Run("Commit the new version string into release")
	stderr, err = git.Checkout(config.Local.Branches.Release)
	if err != nil {
		return failure(stderr, err)
	}

	err = version.Write(nextVersions.Production)
	if err != nil {
		return
	}

	// Step 2: Increment the minor version number and commit it into develop.
	log.V(log.Verbose).Run("Commit the new version string into develop")
	currentTrunk, stderr, err := git.Hexsha(config.Local.Branches.Trunk)
	if err != nil {
		return failure(stderr, err)
	}

	stderr, err = git.Checkout(config.Local.Branches.Trunk)
	if err != nil {
		return failure(stderr, err)
	}
	defer func() {
		if err != nil {
			log.V(log.Verbose).Run("Roll back the trunk branch")
			stderr, ex := git.ResetHard(config.Local.Branches.Trunk, currentTrunk)
			if ex != nil {
				failure(stderr, ex)
			}
		}
	}()

	err = version.Write(nextVersions.Trunk)
	if err != nil {
		return
	}

	// Step 3: Create a new GitHub milestone for the next release.
	if config.Local.Plugins.Milestones {
		log.V(log.Verbose).Run("Create a milestone for the next release")
		err = createMilestone(repoOwner, repoName,
			config.Global.Tokens.GitHub, nextVersions.Production)
		if err != nil {
			return failure(nil, err)
		}
		defer func() {
			if err != nil {
				ex := deleteMilestone(repoOwner, repoName,
					config.Global.Tokens.GitHub, nextVersions.Production)
				if ex != nil {
					failure(nil, ex)
				}
			}
		}()
	}

	// Push the branches.
	err = push()
	return
}

func failure(stderr *bytes.Buffer, err error) error {
	if stderr != nil && stderr.Len() != 0 {
		log.Print(stderr)
	}
	log.Println("Error: ", err)
	return err
}
