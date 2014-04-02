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

package main

import (
	"os"

	"github.com/tchap/trunk/shift"

	"github.com/tchap/gocli"
)

const version = "0.0.1"

func main() {
	// Initialise the application.
	trunk := gocli.NewApp("trunk")
	trunk.UsageLine = "trunk SUBCMD [SUBCMD_OPTION ...]"
	trunk.Short = "Trunk Based Development helper for GitHub"
	trunk.Version = version
	trunk.Long = `
  trunk is a git plugin that provides some useful shortcuts for
  Trunk Based Development on GitHub. See the list of subcommands.`

	// Register subscommands.
	paprika.MustRegisterSubcommand(shift.Command)

	// Run the application.
	paprika.Run(os.Args[1:])
}
