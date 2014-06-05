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

package plugins

import (
	"github.com/tchap/trunk/plugins/circleci"
	"github.com/tchap/trunk/plugins/git"
	"github.com/tchap/trunk/plugins/github"

	"github.com/tchap/go-dwarves/dwarves"
)

// Plugin groups certain domain of workflow functionality together.
type Plugin interface {

	// The plugin name, which is used as a key in the local configuration file.
	Name() string

	// Configuration object factory. This method can return a pointer to
	// a go-yaml-compatible object that is then filled from the configuration file,
	// in this case the one that is saved in the repository.
	//
	// Nil can be returned to signal that no additional config is required.
	NewConfig() interface{}

	// Check is invoked before every command to make sure that
	// the configuration is complete.
	//
	// Nil can be returned to skip this step.
	// ErrDisabled, can be returned to disable the plugin.
	Check(*config.GlobalConfig) *dwarves.Task

	// ReleaseFinish is invoked on `trunk release`.
	ReleaseFinish(*config.GlobalConfig) *dwarves.Task

	// ReleaseStart is invoked on `trunk release`, after ReleaseFinish.
	ReleaseStart(*config.GlobalConfig) *dwarves.Task
}

var Available = [...]Plugin{
	circleci.NewPlugin(),
	git.NewPlugin(),
	github.NewPlugin(),
}
