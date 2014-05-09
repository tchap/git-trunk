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
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	// trunk
	"github.com/tchap/trunk/circleci"
	"github.com/tchap/trunk/config"
	_ "github.com/tchap/trunk/config/autoload"
	"github.com/tchap/trunk/git"
	"github.com/tchap/trunk/log"
	"github.com/tchap/trunk/version"

	// others
	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
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

type result struct {
	msg    string
	stderr *bytes.Buffer
	err    error
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
	stderr, err = git.ResetHard(config.Local.Branches.Production, config.Local.Branches.Release)
	if err != nil {
		log.Print(stderr)
		return
	}

	// Step 6: Commit the new production version string.
	log.V(log.Verbose).Run("Commit the new production version string")
	stderr, err = commitProductionVersion(nextVersions.Production.String())
	if err != nil {
		log.Print(stderr)
		return
	}

	// Step 7: Tag master with the appropriate release tag.
	log.V(log.Verbose).Run("Tag master with the appropriate release tag")
	stderr, err = git.Tag("v" + nextVersions.Production.String())
	if err != nil {
		log.Print(stderr)
		return
	}

	// Step 8: Close the relevant GitHub milestone.
	if config.Local.Plugins.Milestones {
		log.V(log.Verbose).Run("Close the relevant release milestone")
		err = closeMilestone(repoOwner, repoName, config.Global.Tokens.GitHub, versions.Release)
	}
	return
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
	client, err := newGitHubClient(owner, repository, token)
	if err != nil {
		return err
	}

	m, err := getMilestone(client, owner, repository, ver)
	if err != nil {
		return err
	}

	if *m.OpenIssues == 0 {
		return nil
	}
	return fmt.Errorf("milestone not closable: %v", *m.Title)
}

func closeMilestone(owner, repository, token string, ver *version.ReleaseVersion) error {
	client, err := newGitHubClient(owner, repository, token)
	if err != nil {
		return err
	}

	m, err := getMilestone(client, owner, repository, ver)
	if err != nil {
		return err
	}

	_, _, err = client.Issues.EditMilestone(owner, repository, *m.Number, &github.Milestone{
		State: github.String("closed"),
	})
	return err
}

func getMilestone(client *github.Client, owner, repository string, ver *version.ReleaseVersion) (*github.Milestone, error) {
	ms, _, err := client.Issues.ListMilestones(owner, repository, &github.MilestoneListOptions{
		State: "open",
	})
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf("Release %v.%v.%v", ver.Major, ver.Minor, ver.Patch)
	for _, m := range ms {
		if *m.Title == title {
			return &m, nil
		}
	}

	return nil, errors.New("release milestone not found")
}

func newGitHubClient(owner, repository, token string) (*github.Client, error) {
	if token == "" {
		return nil, errors.New("github: token is not set")
	}
	if !regexp.MustCompile("[0-9a-f]{40}").MatchString(token) {
		return nil, errors.New("github: invalid token string")
	}

	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: token},
	}
	return github.NewClient(t.Client()), nil
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
