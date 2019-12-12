package commands

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"

	"github.com/ngageoint/seed-cli/cliutil"
	"github.com/ngageoint/seed-common/util"
)

//DockerList - Simplified version of dockerlist - relies on name filter of
//  docker images command to search for images ending in '-seed'
func DockerList() (string, error) {
	var errs, out bytes.Buffer
	var cmd *exec.Cmd
	reference := util.DockerVersionHasReferenceFilter()
	var buildArgs, dockerCommand = cliutil.DockerCommandArgsInit()
	if reference {
		buildArgs = append(buildArgs, "images", "--filter=reference=*-seed*")
		cmd = exec.Command(dockerCommand, buildArgs...)
	} else {
		buildArgs = append(buildArgs, "images")
		dCmd := exec.Command(dockerCommand, buildArgs...)
		cmd = exec.Command("grep", "-seed")
		var dErr bytes.Buffer
		dCmd.Stderr = &dErr
		dOut, err := dCmd.StdoutPipe()
		if err != nil {
			util.PrintUtil("ERROR: Error attaching to std output pipe. %s\n",
				err.Error())
		}

		dCmd.Start()
		if string(dErr.Bytes()) != "" {
			util.PrintUtil("ERROR: Error reading stderr %s\n",
				string(dErr.Bytes()))
		}

		cmd.Stdin = dOut
	}
	if util.StdErr != nil {
		cmd.Stderr = io.MultiWriter(util.StdErr, &errs)
	} else {
		cmd.Stderr = &errs
	}
	cmd.Stdout = &out

	// run images
	err := cmd.Run()
	if reference && err != nil {
		util.PrintUtil("ERROR: Error executing docker images.\n%s\n",
			err.Error())
		return "", err
	}

	if errs.String() != "" {
		util.PrintUtil("ERROR: Error reading stderr %s\n",
			errs.String())
		return "", errors.New(errs.String())
	}

	if !strings.Contains(out.String(), "seed") {
		util.PrintUtil("No seed images found!\n")
		return "", nil
	}
	util.PrintUtil("%s", out.String())
	return out.String(), nil
}

//PrintListUsage prints the seed list usage information, then exits the program
func PrintListUsage() {
	util.PrintUtil("\nUsage:\tseed list\n")
	util.PrintUtil("\nLists all Seed compliant docker images residing on the local system.\n")
	return
}
