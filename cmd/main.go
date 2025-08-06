/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2024 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: global service admin service entry point
 *
**/

package main

import (
	"fmt"
	"os"

	"github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/utils/sys"

	kingpin "github.com/alecthomas/kingpin/v2"
)

// command line flags
var (
	app = kingpin.New(common.ProgramName, "PETACLOUID Global Service")

	userAPICmd = app.Command("start", fmt.Sprintf("Start the user api service %s", common.ProgramName)).Default()
	versionCmd = app.Command("version", "Show version information")
)

func execute() {

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	// "start" command
	case userAPICmd.FullCommand():
		logger.Infof("Starting user api service %s", metadata.APIVersion.FullVersion())
		err := StartUserAPI()
		if err != nil {
			logger.Panicf("Failed to start %s, %+v", common.GetSvcMetadata().APIVersion.FullVersion(), err)
		}
	case versionCmd.FullCommand():
		fmt.Println(common.GetSvcMetadata().APIVersion.FullVersion())
	}
}

func main() {
	defer sys.Profiling().Stop()

	execute()
}
