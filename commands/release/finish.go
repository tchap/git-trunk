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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"

	// trunk
	"github.com/tchap/trunk/circleci"
	"github.com/tchap/trunk/config"
	"github.com/tchap/trunk/git"
	"github.com/tchap/trunk/log"
	"github.com/tchap/trunk/version"
)

type result struct {
	msg    string
	stderr *bytes.Buffer
	err    error
}

func finishRelease(repoOwner, repoName string, versions, nextVersions *version.Versions) (err error) {
	log.Printf("\n---> Finishing release %v\n", versions.Production)

	// Step 1: Make sure that the workspace and Git index are clean.
	log.V(log.Verbose).Run("Make sure the workspace and Git index are clean")
	if stdout, _, err := git.EnsureCleanWorkingTree(); err != nil {
		log.Print(stdout.String())
		return err
	}

	// Prepare for running the following steps concurrently.
	var numGoroutines int
	results := make(chan *result)

	// Step 2: Make sure that develop, release and master are up to date.
	step2 := &result{msg: "Make sure that all significant branches are synchronized"}
	log.V(log.Verbose).Go(step2.msg)
	numGoroutines++
	go func() {
		step2.stderr, step2.err = checkBranchesInSync()
		results <- step2
	}()

	// Step 3: Make sure that the CI release build is green.
	step3 := &result{msg: "Check the latest release build"}
	if config.Local.Plugins.BuildStatus && !skipBuildCheck {
		log.V(log.Verbose).Go(step3.msg)
		numGoroutines++
		go func() {
			step3.err = checkReleaseBuild(repoOwner, repoName,
				config.Local.Branches.Release, config.Global.Tokens.CircleCi)
			results <- step3
		}()
	}

	// Step 4: Make sure that all the assigned GitHub issues are closed.
	step4 := &result{msg: "Check the relevant release milestone"}
	if config.Local.Plugins.Milestones && !skipMilestoneCheck {
		log.V(log.Verbose).Go(step4.msg)
		numGoroutines++
		go func() {
			step4.err = checkMilestone(repoOwner, repoName,
				config.Global.Tokens.GitHub, versions.Release)
			results <- step4
		}()
	}

	// Wait for the steps goroutines to return and print the results.
	for i := 0; i < numGoroutines; i++ {
		res := <-results
		if res.err == nil {
			log.V(log.Verbose).Ok(res.msg)
			continue
		}

		log.Fail(res.msg)
		if stderr := res.stderr; stderr != nil && stderr.Len() != 0 {
			log.Print(stderr)
		}
		log.Println("Error: ", res.err)
		err = res.err
	}
	if err != nil {
		return ErrActionsFailed
	}

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
	currentBranch, stderr, err := git.CurrentBranch()
	if err != nil {
		log.Print(stderr)
	}
	defer func() {
		log.Run("Check out the original branch")
		stderr, ex := git.Checkout(currentBranch)
		if ex != nil {
			log.Print(stderr)
			if err == nil {
				err = ex
			}
		}
	}()

	// Step 5: Reset master to point to the current release.
	log.V(log.Verbose).Run("Reset master to point to the current release")
	currentProduction, stderr, err := git.Hexsha(config.Local.Branches.Production)
	if err != nil {
		return
	}

	stderr, err = git.ResetHard(config.Local.Branches.Production, config.Local.Branches.Release)
	if err != nil {
		log.Print(stderr)
		return
	}

	// Roll back master in case there is an error encountered later.
	defer func() {
		if err != nil {
			log.V(log.Verbose).Run("Roll back the production branch")
			stderr, ex := git.ResetHard(config.Local.Branches.Production, currentProduction)
			if ex != nil {
				log.Print(stderr)
				log.Println("Error: ", ex)
			}
		}
	}()

	// Step 6: Commit the new production version string.
	log.V(log.Verbose).Run("Commit the new production version string")
	stderr, err = commitProductionVersion(nextVersions.Production.String())
	if err != nil {
		log.Print(stderr)
		return
	}

	// Step 7: Tag master with the appropriate release tag.
	log.V(log.Verbose).Run("Tag master with the appropriate release tag")
	tag := "v" + nextVersions.Production.String()
	stderr, err = git.Tag(tag)
	if err != nil {
		log.Print(stderr)
		return
	}
	// Delete the tag in case there is an error encountered later.
	defer func() {
		if err != nil {
			log.V(log.Verbose).Run("Roll back the release tag")
			stderr, ex := git.DeleteTag(tag)
			if ex != nil {
				log.Print(stderr)
				log.Println("Error: ", ex)
			}
		}
	}()

	// Step 8: Close the relevant GitHub milestone.
	if config.Local.Plugins.Milestones {
		log.V(log.Verbose).Run("Close the relevant release milestone")
		err = closeMilestone(repoOwner, repoName, config.Global.Tokens.GitHub, versions.Release)
	}
	// Re-open the milestone in case there is an error encountered later.
	defer func() {
		if err != nil {
			log.V(log.Verbose).Run("Re-open the relevant release milestone")
			ex := openMilestone(repoOwner, repoName, config.Global.Tokens.GitHub, versions.Release)
			if ex != nil {
				log.Println("Error: ", ex)
			}
		}
	}()

	return startRelease(repoName, repoOwner, versions, nextVersions)
}

func checkBranchesInSync() (stderr *bytes.Buffer, err error) {
	stderr, err = git.Fetch(remote)
	if err != nil {
		return
	}

	bs := [...]string{
		config.Local.Branches.Trunk,
		config.Local.Branches.Release,
		config.Local.Branches.Production,
	}
	for _, b := range bs {
		stderr, err = git.EnsureBranchesEqual(b, remote+"/"+b)
		if err != nil {
			return
		}
	}
	return
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

func checkMilestone(owner, repository, token string, ver *version.ReleaseVersion) error {
	client, err := newGitHubClient(token)
	if err != nil {
		return err
	}

	m, err := getMilestone(client, owner, repository, ver.BaseString())
	if err != nil {
		return err
	}

	if *m.OpenIssues == 0 {
		return nil
	}
	return fmt.Errorf("milestone not closable: %v", *m.Title)
}

func commitProductionVersion(prodVersion string) (stderr *bytes.Buffer, err error) {
	// Checkout master
	stderr, err = git.Checkout(config.Local.Branches.Production)
	if err != nil {
		return
	}

	// Get the absolute path of package.json
	root, stderr, err := git.RepositoryRootAbsolutePath()
	if err != nil {
		return
	}
	path := filepath.Join(root, "package.json")

	// Read package.json
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	// Parse and replace stuff in package.json
	p := regexp.MustCompile(fmt.Sprintf("\"version\": \"%v\"", version.AnyMatcher))
	newContent := p.ReplaceAllLiteral(content,
		[]byte(fmt.Sprintf("\"version\": \"%v\"", prodVersion)))
	if bytes.Equal(content, newContent) {
		err = errors.New("package.json: failed to replace version string")
		return
	}

	// Write package.json
	_, err = file.Seek(0, os.SEEK_SET)
	if err != nil {
		return
	}
	err = file.Truncate(0)
	if err != nil {
		return
	}
	_, err = io.Copy(file, bytes.NewReader(newContent))
	if err != nil {
		return
	}

	// Commit package.json
	_, stderr, err = git.Git("add", path)
	if err != nil {
		return
	}
	_, stderr, err = git.Git("commit", "-m", fmt.Sprintf("Bump version to %v", prodVersion))
	if err != nil {
		return
	}
	return
}
