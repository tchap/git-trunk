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

package shift

import (
	"log"
	"os"

	"github.com/tchap/gocli"
)

var Command = &gocli.Command{
	UsageLine: "shift NEXT_VERSION",
	Short:     "perform the TBD branch shifting operation",
	Long: `
  This subcommand performs the following actions:

    1. Pull develop, release and master.
    2. Close the current GitHub release milestone.
       This operation will fail unless all the assigned issues are closed.
    3. Tag the current release branch.
    4. Reset the master branch to point to the newly created release tag.
    5. Reset the release branch to point to develop (trunk).
    6. If package.json is present in the repository, write the new version
       into the file and commit it into the release branch.
    7. Push the release tag and all the branches (develop, release, master).
    8. Create a new GitHub milestone for the next release.
	`,
	Action: run,
}

func run(cmd *gocli.Command, args []string) {
	log.SetFlags(0)

	// Parse the arguments.
	if len(args) != 1 {
		cmd.Usage()
		os.Exit(2)
	}
}
