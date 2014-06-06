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
	"github.com/tchap/trunk/config"

	"github.com/tchap/go-dwarves/dwarves"
)

type PluginFactory interface {
	PluginName() string
	NewPluginConfig() interface{}
	NewPlugin(pluginConfig interface{}, local *config.Local, global *config.Global) (Plugin, error)
}

type Plugin interface {
	Check() *dwarves.Task
	ReleaseFinish() *dwarves.Task
	ReleaseStart() *dwarves.Task
}
