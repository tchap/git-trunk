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

import (
	"fmt"
	"net/http"
	"strconv"
)

type Project struct {
	client     *Client
	owner      string
	repository string
}

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

type Build struct {
	Status string
}

func (p *Project) Builds(filter *BuildFilter) ([]*Build, *http.Response, error) {
	u := fmt.Sprintf("project/%v/%v", p.owner, p.repository)
	if filter != nil {
		u += "?"
		if v := filter.Branch; v != "" {
			u += "branch=" + v
		}
		if v := filter.Offset; v != 0 {
			u += "offset=" + strconv.Itoa(v)
		}
		if v := filter.Limit; v != 0 {
			u += "limit=" + strconv.Itoa(v)
		}
	}

	req, err := p.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var builds []*Build
	resp, err := p.client.Do(req, &builds)
	if err != nil {
		return nil, resp, err
	}
	return builds, resp, err
}
