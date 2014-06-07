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
	"github.com/tchap/trunk/plugins/plugin"

	"github.com/tchap/go-dwarves/dwarves"
)

type Plugin struct{}

func NewPlugin(localConfigContent, globalConfigContent []byte) (plugin.Plugin, error) {
	return nil, nil
}

func (plugin *Plugin) CheckTask() *dwarves.Task {
	return nil
}

func (plugin *Plugin) ReleaseFinishTask() *dwarves.Task {
	return nil
}

func (plugin *Plugin) ReleaseStartTask() *dwarves.Task {
	return nil
}
