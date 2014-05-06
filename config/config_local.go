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
	"path/filepath"

	"github.com/tchap/trunk/git"

	"gopkg.in/v1/yaml"
)

var Local *LocalConfig

func init() {
	log.SetFlags(0)
	log.Println("Reading the local configuration file...")
	config, err := readLocalConfig()
	if err != nil {
		log.Fatalf("Error: %n\n", err)
	}
	Local = config
}

type LocalConfig struct {
	TrunkBranch       string `yaml:"trunk_branch"`
	ReleaseBranch     string `yaml:"release_branch"`
	ProductionBranch  string `yaml:"production_branch"`
	DisableMilestones bool   `yaml:"disable_milestones"`
	DisableCircleCi   bool   `yaml:"disable_circleci"`
}

func readLocalConfig() (*LocalConfig, error) {
	// Generate the local config file path.
	root, err := git.RepositoryRootAbsolutePath()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, LocalConfigFileName)

	// Read the config file.
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
	var config LocalConfig
	if err := yaml.Unmarshal(content.Bytes(), &config); err != nil {
		return nil, err
	}

	// Fill in the defaults where necessary.
	if config.TrunkBranch == "" {
		config.TrunkBranch = DefaultTrunkBranch
	}
	if config.ReleaseBranch == "" {
		config.ReleaseBranch = DefaultReleaseBranch
	}
	if config.ProductionBranch == "" {
		config.ProductionBranch = DefaultProductionBranch
	}

	// Return the complete LocalConfig instance.
	return &config, nil
}
