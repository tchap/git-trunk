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
	"io"
	"path/filepath"

	"gonuts.org/v1/yaml"
)

type Config struct {
	TrunkBranch       string `yaml:"trunk_branch"`
	ReleaseBranch     string `yaml:"release_branch"`
	ProductionBranch  string `yaml:"production_branch"`
	VersionPattern    string `yaml:"version_pattern"`
	DisableMilestones bool   `yaml:"disable_milestones"`
	DisableCircleCI   bool   `yaml:"disable_circleci"`
}

func ReadConfig(path string) (*Config, error) {
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
	var configFile struct {
		Shift *Config `yaml:"shift"`
	}
	if err := yaml.Unmarshal(content, &configFile); err != nil {
		return nil, err
	}
	config := configFile.Shift

	// Fill in the defaults where necessary.
	if config.TrunkBranch == "" {
		config.TrunkBranch = common.DefaultTrunkBranch
	}
	if config.ReleaseBranch == "" {
		config.ReleaseBranch = common.DefaultReleaseBranch
	}
	if config.ProductionBranch == "" {
		config.ProductionBranch = common.ProductionBranch
	}
	if config.VersionPattern == "" {
		config.VersionPattern = common.DefaultVersionPattern
	}

	// Return the complete Config struct.
	return config.Shift, nil
}

func ReadLocalConfig() (*Config, error) {
	root, err := RepositoryRootAbsolutePath()
	if err != nil {
		return nil, err
	}

	return ReadConfig(filepath.Join(root, common.ConfigFileName))
}
