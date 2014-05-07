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
	"regexp"
	"strconv"

	"github.com/tchap/trunk/config"
	_ "github.com/tchap/trunk/config/autoload"
	"github.com/tchap/trunk/git"
)

const (
	ProductionMatcher = "[0-9]+([.][0-9]+){2}"
	ReleaseMatcher    = ProductionMatcher + "-release"
	TrunkMatcher      = ProductionMatcher + "-dev"
)

type version struct {
	Major uint
	Minor uint
	Patch uint
}

func (v *version) clone() *version {
	return &version{v.Major, v.Minor, v.Patch}
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
		return nil, fmt.Errorf("invalid release version string: %v", raw)
	}
	return &ReleaseVersion{version}, nil
}

func (v *ReleaseVersion) String() string {
	return fmt.Sprintf("%v.%v.%v-release", v.Major, v.Minor, v.Patch)
}

type ProductionVersion struct {
	*version
}

func newProductionVersion(raw string) (*ProductionVersion, error) {
	version, ok := parseVersion(raw, "")
	if !ok {
		return nil, fmt.Errorf("invalid production version string: %v", raw)
	}
	return &ProductionVersion{version}, nil
}

func (v *ProductionVersion) String() string {
	return fmt.Sprintf("%v.%v.%v", v.Major, v.Minor, v.Patch)
}

type Versions struct {
	Trunk      *TrunkVersion
	Release    *ReleaseVersion
	Production *ProductionVersion
}

func LoadVersions() (versions *Versions, stderr *bytes.Buffer, err error) {
	trunkString, stderr, err := readVersion(config.Local.TrunkBranch)
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
	releaseVersion, err := newReleaseVersion(releaseString)
	if err != nil {
		return
	}

	productionString, stderr, err := readVersion(config.Local.ProductionBranch)
	if err != nil {
		return
	}
	productionVersion, err := newProductionVersion(productionString)
	if err != nil {
		return
	}

	versions = &Versions{trunkVersion, releaseVersion, productionVersion}
	return
}

func (vs *Versions) Next(next string) (*Versions, error) {
	var versions *Versions
	if next == "auto" {
		versions = &Versions{
			Trunk:      &TrunkVersion{vs.Trunk.version.clone()},
			Release:    &ReleaseVersion{vs.Trunk.version.clone()},
			Production: &ProductionVersion{vs.Trunk.version.clone()},
		}
		versions.Trunk.Minor++
	} else {
		prodVersion, err := newProductionVersion(next)
		if err != nil {
			return nil, err
		}
		versions := &Versions{
			Trunk:      &TrunkVersion{prodVersion.version.clone()},
			Release:    &ReleaseVersion{prodVersion.version.clone()},
			Production: prodVersion,
		}
		versions.Trunk.Minor++
	}
	return versions, nil
}

func readVersion(branch string) (version string, stderr *bytes.Buffer, err error) {
	// Read package.json
	stdout, stderr, err := git.ShowByBranch(branch, "package.json")
	if err != nil {
		return
	}

	// Parse package.json
	var packageJson struct {
		Version string
	}
	err = json.Unmarshal(stdout.Bytes(), &packageJson)
	if err != nil {
		return
	}

	version = packageJson.Version
	return
}

func parseVersion(versionString, versionSuffix string) (*version, bool) {
	pattern := "^([0-9]+)[.]([0-9]+)[.]([0-9]+)"
	if versionSuffix != "" {
		pattern += "-" + versionSuffix + "$"
	}
	pattern += "$"

	p := regexp.MustCompile(pattern)
	parts := p.FindStringSubmatch(versionString)
	if len(parts) != 4 {
		return nil, false
	}

	major, _ := strconv.ParseUint(parts[1], 10, 32)
	minor, _ := strconv.ParseUint(parts[2], 10, 32)
	patch, _ := strconv.ParseUint(parts[3], 10, 32)

	return &version{uint(major), uint(minor), uint(patch)}, true
}
