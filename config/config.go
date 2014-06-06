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

package config

import (
	"bytes"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/tchap/trunk/git"
)

const (
	LocalConfigFileName  = "trunk.yml"
	GlobalConfigFileName = ".trunk.yml"

	ConfigBranch = "trunk-config"
)

type Local struct {
	Branches struct {
		Trunk      string `yaml:"trunk"`
		Release    string `yaml:"release"`
		Production string `yaml:"production"`
	} `yaml:"branches"`
	Plugins []map[string]interface{} `yaml:"plugins"`
}

func Local(pluginNames []string, pluginConfigs []interface{}) (*Local, error) {
	content, stderr, err := git.ShowByBranch(ConfigBranch, LocalConfigFileName)
	if err != nil {
		log.Fatal(stderr)
		return nil, err
	}

	cfg := Local{Plugins: make([]map[string]interface{})}
	for i := 0; i < len(pluginNames); i++ {
		cfg.Plugins[pluginName[i]] = pluginConfigs[i]
	}
}

func Global() (content *bytes.Buffer, err error) {
	// Generate the global config file path.
	me, err := user.Current()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(me.HomeDir, GlobalConfigFileName)

	// Read the global config file.
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var p bytes.Buffer
	if _, err := io.Copy(&p, file); err != nil {
		return nil, err
	}

	// Return the content.
	return &p, nil
}
