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
	"errors"

	"github.com/tchap/trunk/plugins/plugin"

	"github.com/tchap/go-dwarves/dwarves"
)

func RunCheck(plugins []plugin.Plugin) (err error) {
	// Build the list of check tasks.
	var tasks []dwarves.TaskForest
	for _, p := range plugins {
		tasks = append(tasks, p.CheckTask())
	}

	// Run the check tasks concurrently.
	supervisor := dwarves.NewSupervisor(tasks...)
	monitorCh := make(chan *dwarves.TaskFinishedEvent)
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
	return
}
