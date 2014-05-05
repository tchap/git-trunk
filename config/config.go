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
	"io"
	"path/filepath"

	"gonuts.org/v1/yaml"
)

type GlobalConfig struct {
	CircleCiToken string `yaml:"circleci_token"`
	GitHubToken   string `yaml:"github_token"`
}

// ReadGlobalConfig reads the global configuration file that is expected to be
// placed into the current user's home directory.
//
// When the global config file does not exist, a GlobalConfig instance is returned
// anyway, it just contains the default values where it makes sense.
func ReadGlobalConfig() (*GlobalConfig, error) {
	// Generate the global config file path.
	me, err := user.Current()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(me.HomeDir, common.ConfigFileName)

	// Read the global config file.
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
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
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	// Return the config object.
	return &config, nil
}
