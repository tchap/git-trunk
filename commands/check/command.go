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

package check

import (
	// stdlib
	"bytes"
	"errors"
	"os"

	// trunk
	"github.com/tchap/trunk/config"
	_ "github.com/tchap/trunk/config/autoload"
	"github.com/tchap/trunk/git"
	"github.com/tchap/trunk/log"

	// others
	"github.com/tchap/gocli"
)

var ErrActionsFailed = errors.New("some of the requested actions have failed")

var Command = &gocli.Command{
	UsageLine: "check [-verbose|-debug] [-remote=REMOTE]",
	Short:     "check whether the repository is set up correctly",
	Long: `
  This subcommands analyses the repository to tell the user whether it is
  set up correctly to be used with trunk.
	`,
	Action: run,
}

var (
	verbose bool
	debug   bool
	remote  string = "origin"
)

func init() {
	Command.Flags.BoolVar(&verbose, "v", verbose,
		"print more verbose output")
	Command.Flags.StringVar(&remote, "remote", remote,
		"Git remote to modify")
}

type result struct {
	msg    string
	stderr *bytes.Buffer
	err    error
}

func run(cmd *gocli.Command, args []string) {
	// Parse the arguments.
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(2)
	}

	if verbose {
		log.SetV(log.Verbose)
	}
	if debug {
		log.SetV(log.Debug)
	}

	// Prepare for running the following steps concurrently.
	var numGoroutines int
	results := make(chan *result)

	// Make sure that the relevant branches exist.
	stepB := &result{msg: "Make sure the trunk branch exists"}
	log.V(log.Verbose).Go(stepB.msg)
	numGoroutines++
	go func() {
		_, stepB.stderr, stepB.err = git.Hexsha(config.Local.Branches.Trunk)
		results <- stepB
	}()

	stepC := &result{msg: "Make sure the release branch exists"}
	log.V(log.Verbose).Go(stepC.msg)
	numGoroutines++
	go func() {
		_, stepC.stderr, stepC.err = git.Hexsha(config.Local.Branches.Release)
		results <- stepC
	}()

	stepD := &result{msg: "Make sure the production branch exists"}
	log.V(log.Verbose).Go(stepD.msg)
	numGoroutines++
	go func() {
		_, stepD.stderr, stepD.err = git.Hexsha(config.Local.Branches.Production)
		results <- stepD
	}()

	// Make sure that package.json is present on all the relevant branches.
	stepE := &result{msg: "Make sure package.json exists on the trunk branch"}
	log.V(log.Verbose).Go(stepE.msg)
	numGoroutines++
	go func() {
		_, stepE.stderr, stepE.err = git.ShowByBranch(config.Local.Branches.Trunk, "package.json")
		results <- stepE
	}()

	stepF := &result{msg: "Make sure package.json exists on the release branch"}
	log.V(log.Verbose).Go(stepF.msg)
	numGoroutines++
	go func() {
		_, stepF.stderr, stepF.err = git.ShowByBranch(config.Local.Branches.Release, "package.json")
		results <- stepF
	}()

	stepG := &result{msg: "Make sure package.json exists on the production branch"}
	log.V(log.Verbose).Go(stepG.msg)
	numGoroutines++
	go func() {
		_, stepG.stderr, stepG.err = git.ShowByBranch(config.Local.Branches.Production, "package.json")
		results <- stepG
	}()

	// Wait for the steps goroutines to return and print the results.
	var err error
	for i := 0; i < numGoroutines; i++ {
		res := <-results
		if res.err == nil {
			log.V(log.Verbose).Ok(res.msg)
			continue
		}

		log.Fail(res.msg)
		if stderr := res.stderr; stderr != nil && stderr.Len() != 0 {
			log.V(log.Debug).Print(stderr)
		}
		log.V(log.Debug).Println("Error: ", res.err)
		err = res.err
	}
	if err != nil {
		os.Exit(1)
	}
}
