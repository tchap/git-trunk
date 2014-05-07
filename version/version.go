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

type version struct {
	Major uint
	Minor uint
	Patch uint
}

type TrunkVersion struct {
	*version
}

func newTrunkVersion(raw string) (*TrunkVersion, error) {
	version, ok := parseVersion(raw, "dev")
	if !ok {
		return nil, fmt.Errorf("invalid trunk version string: %v", raw)
	}
	return &TrunkVersion{version}, nil
}

func (v *TrunkVersion) String() string {
	return fmt.Sprintf("%v.%v.%v-dev", v.Major, v.Minor, v.Patch)
}

type ReleaseVersion struct {
	*version
}

func newReleaseVersion(raw string) (*ReleaseVersion, error) {
	version, ok := parseVersion(raw, "release")
	if !ok {
		return nil, fmt.Errorf("invalid trunk version string: %v", raw)
	}
	return &ReleaseVersion{version}, nil
}

func (v *ReleaseVersion) String() string {
	return fmt.Sprintf("%v.%v.%v-release", v.Major, v.Minor, v.Patch)
}

type ProductionVersion version

func newProductionVersion(raw string) (*ProductionVersion, error) {
	version, ok := parseVersion(raw, "release")
	if !ok {
		return nil, fmt.Errorf("invalid trunk version string: %v", raw)
	}
	return (*ProductionVersion)(version), nil
}

type Versions struct {
	Trunk      *TrunkVersion
	Release    *ReleaseVersion
	Production *ProductionVersion
}

func LoadVersions() (versions *Versions, stderr *bytes.Buffer, err error) {
	trunkString, stderr, err = readVersion(config.Local.TrunkBranch)
	if err != nil {
		return
	}
	trunkVersion, err := newTrunkVersion(trunkString)
	if err != nil {
		return
	}

	releaseString, stderr, err := readVersion(config.Local.ReleaseBranch)
	if err != nil {
		return
	}
	releaseVersion, err := newReleaseVersion(trunkString)
	if err != nil {
		return
	}

	productionString, stderr, err := readVersion(config.Local.ProductionBranch)
	if err != nil {
		return
	}
	productionVersion, err := newReleaseVersion(trunkString)
	if err != nil {
		return
	}

	versions = &Versions{trunkVersion, releaseVersion, productionVersion}
	return
}

func (vs *Versions) Next(next string) (versions *Versions, stderr *bytes.Buffer, err error) {

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

func parseVersion(version, versionSuffix string) (*version, bool) {
	pattern := "^([0-9]+)[.]([0-9]+)[.]([0-9]+)"
	if versionSuffix != "" {
		pattern += "-" + versionSuffix
	}
	pattern += "$"

	p := regexp.MustCompile(pattern)
	parts := p.FindStringSubmatch(version)
	if len(parts) != 4 {
		return nil, false
	}

	major, _ := strconv.ParseUint(parts[1], 10, 32)
	minor, _ := strconv.ParseUint(parts[2], 10, 32)
	patch, _ := strconv.ParseUint(parts[3], 10, 32)

	return &versions{major, minor, patch}, true
}
