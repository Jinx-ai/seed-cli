package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ngageoint/seed-cli/cliutil"
	"github.com/ngageoint/seed-cli/constants"
	common_const "github.com/ngageoint/seed-common/constants"
	"github.com/ngageoint/seed-common/util"
)

//Dockerpull pulls specified image from remote repository (default docker.io)
func DockerPull(image, registry, org, username, password string) error {
	if username != "" {
		//set config dir so we don't stomp on other users' logins with sudo
		configDir := common_const.DockerConfigDir + time.Now().Format(time.RFC3339)
		os.Setenv(common_const.DockerConfigKey, configDir)
		defer util.RemoveAllFiles(configDir)
		defer os.Unsetenv(common_const.DockerConfigKey)

		err := util.Login(registry, username, password)
		if err != nil {
			util.PrintUtil(err.Error())
			return err
		}
	}

	if registry == "" {
		registry = common_const.DefaultRegistry
	}

	remoteImage := fmt.Sprintf("%s/%s", registry, image)

	if org != "" {
		remoteImage = fmt.Sprintf("%s/%s/%s", registry, org, image)
	}

	var errs, out bytes.Buffer
	// pull image
	var pullArgs, dockerCommand = cliutil.DockerCommandArgsInit()
	pullArgs = append(pullArgs, "pull", remoteImage)
	util.PrintUtil("INFO: Running Docker command:\ndocker %s\n", strings.Join(pullArgs, " "))
	pullCmd := exec.Command(dockerCommand, pullArgs...)
	if util.StdErr != nil {
		pullCmd.Stderr = io.MultiWriter(util.StdErr, &errs)
	} else {
		pullCmd.Stderr = &errs
	}
	pullCmd.Stdout = &out

	err := pullCmd.Run()
	if err != nil {
		util.PrintUtil("ERROR: Error executing docker pull.\n%s\n",
			err.Error())
		return err
	}

	if errs.String() != "" {
		util.PrintUtil("ERROR: Error reading stderr %s\n",
			errs.String())
		return errors.New(errs.String())
	}

	// tag image
	tagArgs := []string{"tag", remoteImage, image}
	tagCmd := exec.Command("docker", tagArgs...)
	if util.StdErr != nil {
		tagCmd.Stderr = io.MultiWriter(util.StdErr, &errs)
	} else {
		tagCmd.Stderr = &errs
	}
	tagCmd.Stdout = &out

	util.PrintUtil("INFO: Running Docker command:\ndocker %s\n", strings.Join(tagArgs, " "))
	err = tagCmd.Run()
	if err != nil {
		util.PrintUtil("ERROR: Error executing docker tag.\n%s\n",
			err.Error())
		return err
	}

	if errs.String() != "" {
		util.PrintUtil("ERROR: Error reading stderr %s\n",
			errs.String())
		return errors.New(errs.String())
	}

	return nil
}

//PrintPullUsage prints the seed pull usage information, then exits the program
func PrintPullUsage() {
	util.PrintUtil("\nUsage:\tseed pull -in IMAGE_NAME [-r REGISTRY_NAME] [-O ORGANIZATION_NAME] [-u Username] [-p password]\n")
	util.PrintUtil("\nPulls seed image from remote repository.\n")
	util.PrintUtil("\nOptions:\n")
	util.PrintUtil("  -%s -%s Docker image name to pull\n",
		constants.ShortImgNameFlag, constants.ImgNameFlag)
	util.PrintUtil("  -%s  -%s\t Specifies a specific registry (default is index.docker.io).\n",
		constants.ShortRegistryFlag, constants.RegistryFlag)
	util.PrintUtil("  -%s  -%s\t Specifies a specific organization (default is no organization).\n",
		constants.ShortOrgFlag, constants.OrgFlag)
	util.PrintUtil("  -%s  -%s\t Username to login to remote registry (default anonymous).\n",
		constants.ShortUserFlag, constants.UserFlag)
	util.PrintUtil("  -%s  -%s\t Password to login to remote registry (default anonymous).\n",
		constants.ShortPassFlag, constants.PassFlag)
	return
}
