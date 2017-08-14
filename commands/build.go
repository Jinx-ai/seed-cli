package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/ngageoint/seed-cli/util"
	"github.com/ngageoint/seed-cli/constants"
	"github.com/ngageoint/seed-cli/objects"
	"errors"
)

//DockerBuild Builds the docker image with the given image tag.
func DockerBuild(jobDirectory string) (bool, error) {

	seedFileName, err := util.SeedFileName(jobDirectory)
	if err != nil {
		return false, err
	}

	// Validate seed file
	err = ValidateSeedFile("", seedFileName, constants.SchemaManifest)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: seed file could not be validated. See errors for details.")
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		fmt.Fprintf(os.Stderr, "Exiting seed...\n")
		return false, err
	}

	// retrieve seed from seed manifest
	seed := objects.SeedFromManifestFile(seedFileName)

	// Retrieve docker image name
	imageName := objects.BuildImageName(&seed)

	// Build Docker image
	fmt.Fprintf(os.Stderr, "INFO: Building %s\n", imageName)
	buildArgs := []string{"build", "-t", imageName, jobDirectory}
	if util.DockerVersionHasLabel() {
		// Set the seed.manifest.json contents as an image label
		label := "com.ngageoint.seed.manifest=" + objects.GetManifestLabel(seedFileName)
		buildArgs = append(buildArgs, "--label", label)
	}
	cmd := exec.Command("docker", buildArgs...)
	var errs bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &errs)
	cmd.Stdout = os.Stderr

	// Run docker build
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Error executing docker build. %s\n",
			err.Error())
		return false, err
	}

	// check for errors on stderr
	if errs.String() != "" {
		fmt.Fprintf(os.Stderr, "ERROR: Error building image '%s':\n%s\n",
			imageName, errs.String())
		fmt.Fprintf(os.Stderr, "Exiting seed...\n")
		return false, errors.New(errs.String())
	}

	return true, nil
}

//PrintBuildUsage prints the seed build usage arguments, then exits the program
func PrintBuildUsage() {
	fmt.Fprintf(os.Stderr, "\nUsage:\tseed build [-d JOB_DIRECTORY]\n")
	fmt.Fprintf(os.Stderr, "\nOptions:\n")
	fmt.Fprintf(os.Stderr,
		"  -%s  -%s\tDirectory containing Seed spec and Dockerfile (default is current directory)\n",
		constants.ShortJobDirectoryFlag, constants.JobDirectoryFlag)
	os.Exit(0)
}
