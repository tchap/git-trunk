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
	"github.com/tchap/trunk/plugins/circleci"
	"github.com/tchap/trunk/plugins/git"
	"github.com/tchap/trunk/plugins/github"

	"github.com/tchap/go-dwarves/dwarves"
)

// All available plugin factories are registered here.
var factories = [...]PluginFactory{
	circleci.NewPluginFactory(),
	git.NewPluginFactory(),
	github.NewPluginFactory(),
}

func InstantiatePlugins(cfg *config.Global) (ps []Plugin, err error) {
	// Collect available plugin names and their associated config structs.
	var (
		pluginNames   []string
		pluginConfigs []interface{}
	)
	for _, factory := range factories {
		if configStruct := factory.NewPluginConfig(); configStruct != nil {
			pluginNames = append(pluginNames, factory.PluginName())
			pluginConfigs = append(pluginConfigs, configStruct)
		}
	}

	// Feed the config structs from the local configuration file.
	localConfig, err := config.Local(pluginNames, pluginConfigs)
	if err != nil {
		return nil, err
	}

	// Read global config as well.
	globalConfig, err := config.Global()
	if err != nil {
		return nil, err
	}

	// Build the list of enabled plugins.
	var plugins []Plugin
	for i, factory := range factory {
		plugin, err := factory.NewPlugin(pluginConfigs[i], localConfig, globalConfig)
		if err != nil {
			return nil, err
		}
		if plugin != nil {
			plugins = append(plugins, plugin)
		}
	}

	// Run the Check tasks.
	var tasks []*dwarves.Task
	for _, plugin := range plugins {
		tasks = append(tasks, plugin.CheckTask())
	}
	supervisor := dwarves.NewSupervisor(tasks...)
	monitorCh := make(chan *dwarves.TaskError)
	if err := supervisor.DispatchDwarves(monitorCh); err != nil {
		panic(err)
	}
	for {
		event, ok := <-monitorCh
		if !ok {
			break
		}
		if event.Error != nil {
			err = errors.New("failed to initialise plugins")
		}
	}

	// Return the list of enabled plugins in case all the checks passed.
	if err == nil {
		ps = plugins
	}
	return
}
