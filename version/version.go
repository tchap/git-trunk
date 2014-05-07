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

package version

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/tchap/trunk/config"
	_ "github.com/tchap/trunk/config/autoload"
	"github.com/tchap/trunk/git"
)

const FullVersionPattern = "[0-9]+([.][0-9]+){2}"

type version struct {
	Major uint
	Minor uint
	Patch uint
}

type TrunkVersion struct {
	*version
}

func (v *TrunkVersion) String() string {
	return fmt.Sprintf("%v.%v.%v-dev", v.Major, v.Minor, v.Patch)
}

type ReleaseVersion struct {
	*version
}

func (v *ReleaseVersion) String() string {
	return fmt.Sprintf("%v.%v.%v-release", v.Major, v.Minor, v.Patch)
}

type ProductionVersion version

type Versions struct {
	TrunkCurrent      *TrunkVersion
	TrunkNext         *TrunkVersion
	ReleaseCurrent    *ReleaseVersion
	ReleaseNext       *ReleaseVersion
	ProductionCurrent *ProductionVersion
	ProductionNext    *ProductionVersion
}

func ComputeVersions(next string) (vs *Versions, stderr *bytes.Buffer, err error) {
	trunkCurrent, stderr, err = readVersion(config.Local.TrunkBranch)
	if err != nil {
		return
	}
	// Fill TrunkCurrent.
	versions := &Versions{
		TrunkCurrent: trunkCurrent,
	}

	if next == "auto" {
		next = regexp.MustCompile("^" + FullVersionPattern).FindString(trunkCurrent)
		if next == "" {
			err = fmt.Errorf("branch %v: package.json: malformed version string: %v",
				config.Local.TrunkBranch, next)
		}
	} else {
		if !regexp.MustCompile("^" + FullVersionPattern + "$").Match(next) {
			err = fmt.Errorf("malformed version string: %v", next)
		}
	}
	// Fill ReleaseNext.
	versions.ReleaseNext = next

}

func readVersion(branch string) (version string, stderr *bytes.Buffer, err error) {
	// Read package.json
	stdout, stderr, err := git.ShowByBranch(branch, "package.json")
	if err != nil {
		return err
	}

	// Parse package.json
	var packageJson struct {
		Version string
	}
	err = json.Unmarshal(stdout.Bytes(), &packageJson)
	if err != nil {
		return
	}

	return packageJson.Version, nil
}
