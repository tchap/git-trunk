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
	"log"
	"os"
	"os/user"
	"path/filepath"

	"gopkg.in/v1/yaml"
)

var Global *GlobalConfig

type GlobalConfig struct {
	CircleCiToken string `yaml:"circleci_token"`
	GitHubToken   string `yaml:"github_token"`
}

func init() {
	log.SetFlags(0)
	log.Println("Reading the global configuration file...")
	config, err := readGlobalConfig()
	if err != nil {
		log.Fatalf("Error: %n\n", err)
	}
	Global = config
}

func readGlobalConfig() (*GlobalConfig, error) {
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
			return &GlobalConfig{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var content bytes.Buffer
	if _, err := io.Copy(&content, file); err != nil {
		return nil, err
	}

	// Parse the content.
	var config GlobalConfig
	if err := yaml.Unmarshal(content.Bytes(), &config); err != nil {
		return nil, err
	}

	// Return the config object.
	return &config, nil
}
