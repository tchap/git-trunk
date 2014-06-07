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
	"github.com/tchap/trunk/log"
	"github.com/tchap/trunk/plugins/circleci"
	"github.com/tchap/trunk/plugins/git"
	"github.com/tchap/trunk/plugins/github"
	"github.com/tchap/trunk/plugins/plugin"
)

// All available plugin factories are listed here.
var factories = [...]plugin.Factory{
	circleci.NewPlugin,
	git.NewPlugin,
	github.NewPlugin,
}

func GetEnabledPlugins() (enabledPlugins []plugin.Plugin, err error) {
	// Read local config.
	localConfigContent, stderr, err := config.ReadLocalConfig()
	if err != nil {
		log.Fatalln(stderr)
		return nil, err
	}

	// Read global config.
	globalConfigContent, err := config.ReadGlobalConfig()
	if err != nil {
		log.Fatalln(stderr)
		return nil, err
	}

	// Build the list of enabled plugins.
	var plugins []plugin.Plugin
	for _, factory := range factories {
		plugin, err := factory(localConfigContent.Bytes(), globalConfigContent.Bytes())
		if err != nil {
			return nil, err
		}
		if plugin != nil {
			plugins = append(plugins, plugin)
		}
	}

	return plugins, nil
}
