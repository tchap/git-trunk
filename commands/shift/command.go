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
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	//	"os/signal"

	"github.com/tchap/trunk/circleci"
	"github.com/tchap/trunk/config"
	_ "github.com/tchap/trunk/config/autoload"
	"github.com/tchap/trunk/git"
	"github.com/tchap/trunk/log"
	"github.com/tchap/trunk/version"

	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
	"github.com/tchap/gocli"
)

var ErrActionsFailed = errors.New("some of the requested actions have failed")

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
    2. Make sure that develop, release and master are up to date.
    3. Make sure that the release CI builds are green.
    4. Make sure that all the assigned GitHub issues are closed.
    5. Reset master to point to the current release.
    6. Commit the new production version string.
    7. Tag master with the appropriate release tag.
    8. Close the relevant GitHub milestone.

  Initialization of the next release:
    1. In case the next version is specified manually, commit it into develop.
    2. Reset release to point to develop and commit the new version string.
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
	// Parse the arguments.
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	// Perform the shifting.
	if err := shift(next); err != nil {
		log.Fatal("\nFatal: ", err)
	}
}

type result struct {
	msg    string
	stderr *bytes.Buffer
	err    error
}

func shift(next string) (err error) {
	// Parse the relevant Git remote to get the GitHub repository name and owner.
	log.Run("Read the GitHub repository name and owner")
	repoOwner, repoName, stderr, err := getGitHubOwnerAndRepository()
	if err != nil {
		log.Print(stderr.String())
		return
	}

	// Read the current version strings.
	log.Run("Load the current version strings")
	versions, stderr, err := version.LoadVersions()
	if err != nil {
		log.Print(stderr.String())
		return
	}

	// Step 1: Make sure that the workspace and Git index are clean.
	log.Run("Check the workspace and Git index")
	if stdout, _, err := git.EnsureCleanWorkingTree(); err != nil {
		log.Print(stdout.String())
		return err
	}

	// Prepare for running the following steps concurrently.
	results := make(chan *result, 3)

	// Step 2: Make sure that develop and release are up to date.
	step2 := &result{msg: "Check whether the local and remote refs are synchronized"}
	log.Go(step2.msg)
	go func() {
		stderr, err := git.Fetch(remote)
		if err != nil {
			goto Exit
		}

		stderr, err = git.EnsureBranchesEqual(
			config.Local.TrunkBranch, remote+"/"+config.Local.TrunkBranch)
		if err != nil {
			goto Exit
		}

		stderr, err = git.EnsureBranchesEqual(
			config.Local.ReleaseBranch, remote+"/"+config.Local.ReleaseBranch)
		if err != nil {
			goto Exit
		}

		stderr, err = git.EnsureBranchesEqual(
			config.Local.ProductionBranch, remote+"/"+config.Local.ProductionBranch)
		if err != nil {
			goto Exit
		}

	Exit:
		step2.stderr = stderr
		step2.err = err
		results <- step2
		return
	}()

	// Step 3: Make sure that the CI release build is green.
	step3 := &result{msg: "Check the latest release build"}
	if !skipBuildCheck && !config.Local.DisableCircleCi {
		log.Go(step3.msg)
		go func() {
			step3.err = checkReleaseBuild(repoOwner, repoName,
				config.Local.ReleaseBranch, config.Global.CircleCiToken)
			results <- step3
		}()
	} else {
		log.Skip(step3.msg)
		results <- step3
	}

	// Step 4: Make sure that all the assigned GitHub issues are closed.
	step4 := &result{msg: "Check the relevant release milestone"}
	if !skipMilestoneCheck && !config.Local.DisableMilestones {
		log.Go(step4.msg)
		go func() {
			step4.err = checkMilestone(repoOwner, repoName,
				versions.Release, config.Global.GitHubToken)
			results <- step4
		}()
	} else {
		log.Skip(step4.msg)
		results <- step4
	}

	// Wait for the steps goroutines to return and print the results.
	for i := 0; i < cap(results); i++ {
		res := <-results
		if res.err == nil {
			log.Ok(res.msg)
			continue
		}

		log.Fail(res.msg)
		if stderr := res.stderr; stderr != nil && stderr.Len() != 0 {
			log.Print(stderr.String())
		}
		log.Println("Error: ", res.err)
		err = res.err
	}
	if err != nil {
		return ErrActionsFailed
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

func checkReleaseBuild(owner, repository, branch, token string) error {
	// Fetch the latest release build from Circle CI.
	circle, err := circleci.NewClient(token)
	if err != nil {
		return err
	}

	builds, _, err := circle.Project(owner, repository).Builds(&circleci.BuildFilter{
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

func checkMilestone(owner, repository string, ver *version.ReleaseVersion, token string) error {
	if token == "" {
		return errors.New("github: token is not set")
	}
	if !regexp.MustCompile("[0-9a-f]{40}").MatchString(token) {
		return errors.New("github: invalid token string")
	}

	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: token},
	}
	c := github.NewClient(t.Client())
	ms, _, err := c.Issues.ListMilestones(owner, repository, &github.MilestoneListOptions{
		State: "open",
	})
	if err != nil {
		return err
	}

	title := fmt.Sprintf("Release %v.%v.%v", ver.Major, ver.Minor, ver.Patch)
	for _, m := range ms {
		if *m.Title == title {
			if *m.OpenIssues == 0 {
				return nil
			}
			return fmt.Errorf("milestone not closable: %v", title)
		}
	}
	return fmt.Errorf("milestone not found: %v", title)
}

func getGitHubOwnerAndRepository() (owner, repository string, stderr *bytes.Buffer, err error) {
	// Get the GitHub owner and repository from the relevant remote URL.
	var stdout bytes.Buffer
	stderr = new(bytes.Buffer)
	cmd := exec.Command("git", "config", "--get", "remote."+remote+".url")
	cmd.Stdout = &stdout
	cmd.Stderr = stderr

	err = cmd.Run()
	if err != nil {
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
