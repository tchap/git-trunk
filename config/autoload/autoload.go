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

package autoload

import (
	"github.com/tchap/trunk/config"
	"github.com/tchap/trunk/log"
)

// Read the global configuration file and save it into config.Global.
func init() {
	log.V(log.Verbose).Run("Read the global configuration file")
	cfg, err := config.ReadGlobalConfig()
	if err != nil {
		log.Fatalf("Error: %n\n", err)
	}
	config.Global = cfg
}

// Read the local configuration file and save it into config.Local.
func init() {
	log.V(log.Verbose).Run("Read the local configuration file")
	cfg, err := config.ReadLocalConfig()
	if err != nil {
		log.Fatalf("Error: %n\n", err)
	}
	config.Local = cfg
}
