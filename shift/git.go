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
	"time"

	"github.com/libgit2/git2go"
)

func getUserSignature(repo *git.Repository) (*git.Signature, error) {
	config, err := repo.Config()
	if err != nil {
		return nil, err
	}

	name, err := config.LookupString("user.name")
	if err != nil {
		return nil, err
	}

	email, err := config.LookupString("user.email")
	if err != nil {
		return nil, err
	}

	return &git.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}, nil
}

func setCallbacks(remote *git.Remote) {
	remote.SetCallbacks(&git.RemoteCallbacks{
		CredentialsCallback: credentialsCallback,
	})
}

func credentialsCallback(url, username string, allowedTypes git.CredType) (int, *git.Cred) {
	ret, cred := git.NewCredSshKeyFromAgent(username)
	return ret, &cred
}

func checkout(repo *git.Repository, branchName string) (*git.Branch, error) {
	branch, err := repo.LookupBranch(branch, git.BranchLocal)
	if err != nil {
		return nil, err
	}

}
