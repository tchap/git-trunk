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

package circleci

func (c *Client) Project(owner, repository string) *Project {
	return &Project{
		client:     c,
		owner:      owner,
		repository: repository,
	}
}

type BuildFilter struct {
	Branch string
	Offset int
	Limit  int
}

func (p *Project) Builds(filter *BuildFilter) ([]*Builds, *http.Response, error) {

}
