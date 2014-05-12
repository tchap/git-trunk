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
	"net/url"
	"os/exec"
	"regexp"
	"strings"

	// trunk
	"github.com/tchap/trunk/version"

	// others
	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
)

func newGitHubClient(token string) (*github.Client, error) {
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

func getMilestone(client *github.Client, owner, repository, version string) (*github.Milestone, error) {
	ms, _, err := client.Issues.ListMilestones(owner, repository, &github.MilestoneListOptions{
		State: "open",
	})
	if err != nil {
		return nil, err
	}

	title := "Release " + version
	for _, m := range ms {
		if *m.Title == title {
			return &m, nil
		}
	}

	return nil, errors.New("release milestone not found")
}

func createMilestone(owner, repository, token string, ver *version.ProductionVersion) error {
	client, err := newGitHubClient(token)
	if err != nil {
		return err
	}

	title := fmt.Sprintf("Release %v", ver)
	_, _, err = client.Issues.CreateMilestone(owner, repository, &github.Milestone{
		Title: &title,
	})
	return err
}

func closeMilestone(owner, repository, token string, ver *version.ReleaseVersion) error {
	return setMilestoneStatus(owner, repository, token, ver, "closed")
}

func openMilestone(owner, repository, token string, ver *version.ReleaseVersion) error {
	return setMilestoneStatus(owner, repository, token, ver, "open")
}

func setMilestoneStatus(owner, repository, token string, ver *version.ReleaseVersion, status string) error {
	client, err := newGitHubClient(token)
	if err != nil {
		return err
	}

	m, err := getMilestone(client, owner, repository, ver.BaseString())
	if err != nil {
		return err
	}

	_, _, err = client.Issues.EditMilestone(owner, repository, *m.Number, &github.Milestone{
		State: github.String(status),
	})
	return err
}

func deleteMilestone(owner, repository, token string, ver *version.ProductionVersion) error {
	client, err := newGitHubClient(token)
	if err != nil {
		return err
	}

	m, err := getMilestone(client, owner, repository, ver.BaseString())
	if err != nil {
		return err
	}

	_, err = client.Issues.DeleteMilestone(owner, repository, *m.Number)
	return err
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
