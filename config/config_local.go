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
	"path/filepath"

	"github.com/tchap/trunk/git"

	"gopkg.in/v1/yaml"
)

var Local *LocalConfig

type LocalConfig struct {
	Branches struct {
		Trunk      string `yaml:"trunk"`
		Release    string `yaml:"release"`
		Production string `yaml:"production"`
	} `yaml:"branches"`
	Plugins struct {
		Milestones  bool `yaml:"github_milestones"`
		BuildStatus bool `yaml:"circleci_status"`
	} `yaml:"plugins"`
}

func ReadLocalConfig() (*LocalConfig, error) {
	// Generate the local config file path.
	root, _, err := git.RepositoryRootAbsolutePath()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, LocalConfigFileName)

	// Read the config file.
	var (
		config  LocalConfig
		content bytes.Buffer
	)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			goto Exit
		}
		return nil, err
	}
	defer file.Close()

	if _, err := io.Copy(&content, file); err != nil {
		return nil, err
	}

	// Parse the content.
	if err := yaml.Unmarshal(content.Bytes(), &config); err != nil {
		return nil, err
	}

	// Fill in the defaults where necessary.
Exit:
	if config.Branches.Trunk == "" {
		config.Branches.Trunk = DefaultTrunkBranch
	}
	if config.Branches.Release == "" {
		config.Branches.Release = DefaultReleaseBranch
	}
	if config.Branches.Production == "" {
		config.Branches.Production = DefaultProductionBranch
	}

	// Return the complete LocalConfig instance.
	return &config, nil
}
