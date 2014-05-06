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

func Reset(branch, hexsha string) (stderr *bytes.Buffer, err error) {
	stderr, err = execCommand("git", "checkout", branch)
	if err != nil {
		return
	}

	return execCommand("git", "reset", "--hard", hexsha)
}

func Hexsha(ref string) (hexsha string, stderr *bytes.Buffer, err error) {
	var stdout bytes.Buffer
	stderr = new(bytes.Buffer)
	cmd := exec.Command("git", "rev-parse", ref)
	cmd.Stdout = &stdout
	cmd.Stderr = stderr

	err = cmd.Run()
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

func EnsureCleanWorkspaceAndIndex() (status *bytes.Buffer, stderr *bytes.Buffer, err error) {
	status = new(bytes.Buffer)
	stderr = new(bytes.Buffer)
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Stdout = status
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return
	}
	if status.Len() != 0 {
		err = ErrDirtyRepository
	}
	return
}

func Fetch(remote string) (stderr *bytes.Buffer, err error) {
	return execCommand("git", "fetch", remote)
}

func execCommand(name string, args ...string) (stderr *bytes.Buffer, err error) {
	stderr = new(bytes.Buffer)
	cmd := exec.Command(name, args...)
	cmd.Stderr = stderr
	err = cmd.Run()
	return
}
