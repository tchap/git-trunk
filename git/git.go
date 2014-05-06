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

package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

var ErrDirtyRepository = errors.New("the repository is dirty")

func Fetch(remotes ...string) (stderr *bytes.Buffer, err error) {
	args := append([]string{"fetch"}, remotes...)
	_, stderr, err = Git(args...)
	return
}

func Checkout(branch string) (stderr *bytes.Buffer, err error) {
	_, stderr, err = Git("checkout", branch)
	return
}

func Reset(branch, ref string) (stderr *bytes.Buffer, err error) {
	stderr, err = Checkout(branch)
	if err != nil {
		return
	}

	_, stderr, err = Git("reset", "--hard", ref)
	return
}

func Hexsha(ref string) (hexsha string, stderr *bytes.Buffer, err error) {
	stdout, stderr, err := Git("rev-parse", ref)
	if err != nil {
		return
	}

	hexsha = string(bytes.TrimSpace(stdout.Bytes()))
	return
}

func EnsureBranchesEqual(b1, b2 string) (stderr *bytes.Buffer, err error) {
	hexsha1, stderr, err := Hexsha(b1)
	if err != nil {
		return
	}
	hexsha2, stderr, err := Hexsha(b2)
	if err != nil {
		return
	}

	if hexsha1 != hexsha2 {
		err = fmt.Errorf("branches %v and %v need merging", b1, b2)
	}
	return
}

func EnsureCleanWorkingTree() (status *bytes.Buffer, stderr *bytes.Buffer, err error) {
	status, stderr, err = Git("status", "--porcelain")
	if status.Len() != 0 {
		err = ErrDirtyRepository
	}
	return
}

func RepositoryRootAbsolutePath() (path string, stderr *bytes.Buffer, err error) {
	stdout, stderr, err := Git("rev-parse", "--show-toplevel")
	if err != nil {
		return
	}

	path = string(bytes.TrimSpace(stdout.Bytes()))
	return
}

func Git(args ...string) (stdout, stderr *bytes.Buffer, err error) {
	stdout = new(bytes.Buffer)
	stderr = new(bytes.Buffer)
	cmd := exec.Command("git", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	return
}
