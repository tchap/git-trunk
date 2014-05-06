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
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	//	"os/signal"
	"regexp"

	"github.com/tchap/trunk/config"
	_ "github.com/tchap/trunk/config/autoload"
	"github.com/tchap/trunk/git"

	"github.com/tchap/gocli"
)

const VersionPattern = "[0-9]+(.[0-9+])?"

var Command = &gocli.Command{
	UsageLine: `
  shift [-remote=REMOTE] [-next=NEXT]
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
    2. Make sure that develop and release are up to date.
    3. Make sure that the release CI builds are green.
    4. Make sure that all the assigned GitHub issues are closed.
    5. Read the current release version string from package.json, then reset
       master to point to the current release. Format the version string and
       commit it into master.
    6. Tag master as the new production release.
    7. Close the relevant GitHub milestone.

  Initialization of the next release:
    1. In case the next version is specified manually, commit it into develop.
    2. Reset release to point to develop.
    3. Increment the minor version number and commit it into develop.
    4. Create a new GitHub milestone for the next release.

  Finalization:
    1. Push the release tag and all the branches (develop, release, master).
	`,
	Action: run,
}

var (
	remote             string = "origin"
	next               string = "auto"
	skipMilestoneCheck bool
	skipBuildCheck     bool
)

func init() {
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
	log.SetFlags(0)

	// Parse the arguments.
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	if next != "auto" {
		matched, err := regexp.Match(VersionPattern, []byte(next))
		if err != nil {
			log.Fatal("\nError: ", err)
		}
		if !matched {
			log.Fatal("\nError: Invalid version string")
		}
	}

	// Perform the shifting.
	if err := shift(next); err != nil {
		log.Fatal("\nError: ", err)
	}
}

type result struct {
	step   int
	stderr *bytes.Buffer
	err    error
}

func shift(next string) (err error) {
	// Parse the relevant Git remote to get the GitHub repository name and owner.
	log.Println("[RUN]  Read the GitHub repository name and owner")
	repoOwner, repoName, err := getGitHubOwnerAndRepository()
	if err != nil {
		return
	}

	// Step 1: Make sure that the workspace and Git index are clean.
	log.Println("[RUN]  Check the workspace and Git index")
	if stdout, _, err := git.EnsureCleanWorkingTree(); err != nil {
		log.Print(stdout.String())
		return err
	}

	// Prepare for running the following steps concurrently.
	const (
		step2 = iota
		step3
		step4
	)
	messages := [...]string{
		"Check whether the local and remote refs are synchronized",
		"Check the latest release build",
		"Check the relevant release milestone",
	}
	results := make(chan *result, 3)

	// Step 2: Make sure that develop and release are up to date.
	log.Printf("[GO]   %v\n", messages[step2])
	go func() {
		if stderr, err := git.Fetch(remote); err != nil {
			results <- &result{step2, stderr, err}
			return
		}

		stderr, err := git.EnsureBranchesEqual(
			config.Local.TrunkBranch, remote+"/"+config.Local.TrunkBranch)
		if err != nil {
			results <- &result{step2, stderr, err}
			return
		}
		stderr, err = git.EnsureBranchesEqual(
			config.Local.ReleaseBranch, remote+"/"+config.Local.ReleaseBranch)
		if err != nil {
			results <- &result{step2, stderr, err}
			return
		}
	}()

	// Step 3: Make sure that the CI release build is green.
	if !skipBuildCheck && !config.Local.DisableCircleCi {
		log.Printf("[GO]   %v\n", messages[step3])
		go func() {
			err := checkReleaseBuild(repoOwner, repoName,
				config.Local.ReleaseBranch, config.Global.CircleCiToken)
			results <- &result{step3, nil, err}
		}()
	} else {
		log.Printf("[SKIP] %v\n", messages[step3])
		results <- nil
	}

	// Step 4: Make sure that all the assigned GitHub issues are closed.
	if !skipMilestoneCheck && !config.Local.DisableMilestones {
		log.Printf("[GO]   %v\n", messages[step4])
		go func() {
			err = checkMilestone(repoOwner, repoName, versions.ReleaseCurrent)
			results <- &result{step4, nil, err}
		}()
	} else {
		log.Printf("[SKIP] %v\n", messages[step4])
		results <- nil
	}

	// Wait for the steps goroutines to return and print the results.
	for i := 0; i < cap(results); i++ {
		res := <-results
		if res == nil {
			continue
		}
		if res.err == nil {
			log.Printf("[OK]   %v\n", messages[i])
			continue
		}

		log.Printf("[FAIL] %v\n", messages[i])
		if res.stderr != nil {
			log.Print(res.stderr.String())
		}
		log.Println("\nError: ", res.err)
		err = res.err
	}
	if err != nil {
		return
	}

	/*
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

		// Make sure the same branch is checked out when we are done.
		output, err = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			return
		}
		currentBranch := string(bytes.TrimSpace(output))
		defer func() {
			log.Printf("---> Checking out the original branch (%v)\n", currentBranch)
			output, ex := exec.Command("git", "checkout", currentBranch).CombinedOutput()
			if ex != nil {
				log.Print(output)
				if err == nil {
					err = ex
				}
			}
		}()

		// Rollback develop and release in case something goes wrong.
		trunkHexsha, err := hexsha(TrunkBranch)
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				reset(TrunkBranch, trunkHexsha)
			}
		}()

		releaseHexsha, err := hexsha(ReleaseBranch)
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				reset(ReleaseBranch, releaseHexsha)
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
	*/
	return nil
}

func checkReleaseBuild(owner, repository, branch, circleCiToken string) error {
	// Make sure the Circle CI token is specified in the config.
	if circleCiToken == "" {
		return newErrGlobalConfig("circleci_token")
	}

	// Fetch the latest release build from Circle CI.
	project := circleci.NewClient(circleCiToken).Project(owner, repository)
	builds, err := project.Builds(&circleci.BuildFilter{
		Branch: branch,
		Limit:  1,
	})
	if err != nil {
		return err
	}

	// Check the build results.
	if len(builds) != 1 {
		return fmt.Errorf("No build found (owner=%v, repo=%v, branch=%v)",
			owner, repository, branch)
	}
	if stat := builds[0].Status; stat != "success" {
		return fmt.Errorf("The release build is not passing (status=%v)", stat)
	}
	return nil
}

func getGitHubOwnerAndRepository() (owner, repository string, err error) {
	// Get the GitHub owner and repository from the relevant remote URL.
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd := exec.Command("git", "config", "--get", "remote."+remote+"url")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		log.Print(stderr.String())
		return
	}

	ru := string(bytes.TrimSpace(stdout.Bytes()))
	ru = strings.Replace(ru, ":", "/", -1)
	if strings.HasSuffix(ru, ".git") {
		ru = ru[:len(ru)-4]
	}
	repoURL, err := url.Parse(ru)
	if err != nil {
		return
	}

	parts := strings.Split(repoURL.Path, "/")
	if len(parts) != 3 {
		err = fmt.Errorf("Invalid GitHub remote URL: %v", ru)
		return
	}

	owner = parts[1]
	repository = parts[2]
	return
}
