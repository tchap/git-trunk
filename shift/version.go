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

type Versions struct {
	TrunkCurrent      string
	TrunkNext         string
	ReleaseCurrent    string
	ReleaseNext       string
	ProductionCurrent string
	ProductionNext    string
}

func ComputeVersions(next string) (vs *Versions, stderr *bytes.Buffer, err error) {
	if next == "auto" {
		stderr, err = git.Checkout(trunkBranch)
		if err != nil {
			return
		}

		next, err := readVersion()
		if err != nil {
			return
		}

	}
	return computeNextVersions(next)
}

func computeVersions(next string) (versions *Versions, stderr *bytes.Buffer, err error) {

}
