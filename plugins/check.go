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
	"github.com/tchap/trunk/config"
	"github.com/tchap/trunk/plugins/git"
	"github.com/tchap/trunk/plugins/github"

	"github.com/tchap/go-dwarves/dwarves"
)

func InstantiatePlugins(cfg *config.Global) (ps []Plugin, err error) {
	var available = [...]Plugin{
		circleci.NewPlugin(),
		git.NewPlugin(),
		github.NewPlugin(),
	}

	var (
		tasks   []*dwarves.Task
		enabled []Plugin
	)
	for _, plugin := range available {
		if task := plugin.CheckTask(cfg); task != nil {
			tasks = append(tasks, task)
		}
	}

	localConfig := config.NewLocal()
	for _, plugin := range available {
		var (
			pluginName   = plugin.Name()
			pluginConfig = plugin.NewConfig()
		)
		if _, ok := localConfig.Plugins[pluginName]; !ok {
			panic(fmt.Errorf("Plugin name clash: %v", pluginName))
		}
		localConfig.Plugins[pluginName] = pluginConfig
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
}
