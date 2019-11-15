package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ngageoint/seed-cli/constants"
	common_const "github.com/ngageoint/seed-common/constants"
	"github.com/ngageoint/seed-common/objects"
	RegistryFactory "github.com/ngageoint/seed-common/registry"
	"github.com/ngageoint/seed-common/util"
)

//DockerPublish executes the seed publish command
func DockerPublish(origImg, manifest, registry, org, username, password, jobDirectory string,
	force, P, pm, pp, J, jm, jp bool) (string, error) {

	if origImg == "" {
		util.PrintUtil("INFO: Image name not specified. Attempting to use manifest: %v\n", manifest)
		temp, err := objects.GetImageNameFromManifest(manifest, jobDirectory)
		if err != nil {
			return "", err
		}
		origImg = temp
	}

	if origImg == "" {
		err := errors.New("ERROR: No input image specified.")
		util.PrintUtil("%s\n", err.Error())
		return "", err
	}

	if exists, err := util.ImageExists(origImg); !exists {
		if err != nil {
			util.PrintUtil("%s\n", err.Error())
			return "", err
		}
		msg := fmt.Sprintf("Unable to find image: %s. Did you specify a valid tag?", origImg)
		util.PrintUtil("%s\n", msg)
		return "", errors.New(msg)
	}

	temp := strings.Split(origImg, ":")
	if len(temp) != 2 {
		err := errors.New("ERROR: Invalid seed name: %s. Unable to split into name/tag pair\n")
		return "", err
	}
	repoName := temp[0]
	repoTag := temp[1]

	if username != "" {
		//set config dir so we don't stomp on other users' logins with sudo
		configDir := common_const.DockerConfigDir + time.Now().Format(time.RFC3339)
		os.Setenv(common_const.DockerConfigKey, configDir)
		defer util.RemoveAllFiles(configDir)
		defer os.Unsetenv(common_const.DockerConfigKey)

		err := util.Login(registry, username, password)
		if err != nil {
			util.PrintUtil(err.Error())
		}
	}

	//1. Check names and verify it doesn't conflict
	tag := ""
	img := origImg

	// docker tag if registry and/or org specified
	if registry != "" || org != "" {
		if org != "" {
			tag = org + "/"
			repoName = tag + repoName
		}
		if registry != "" {
			tag = registry + "/" + tag
		}

		img = tag + img
	}

	// Check for image confliction.
	conflict := false
	if !force {
		reg, err := RegistryFactory.CreateRegistry(registry, org, username, password)
		if err != nil {
			err = errors.New(checkError(err, registry, username, password))
			return "", err
		}
		if reg == nil {
			err = errors.New("Unknown error connecting to registry")
			return "", err
		}

		if reg != nil && err == nil {
			manifest, _ := reg.GetImageManifest(repoName, repoTag)
			conflict = manifest != ""
		}

		if conflict {
			util.PrintUtil("INFO: Image %s exists on registry %s\n", img, registry)
		}
	}

	// If it conflicts, bump specified version number
	if conflict && !force {
		util.PrintUtil("INFO: Force flag not specified, attempting to rebuild with new version number.\n")

		//1. Verify we have a valid manifest
		seedFileName := ""
		if manifest != "." && manifest != "" {
			seedFileName = util.GetFullPath(manifest, jobDirectory)
			if _, err := os.Stat(seedFileName); os.IsNotExist(err) {
				util.PrintUtil("ERROR: Seed manifest not found. %s\n", err.Error())
				return "", err
			}
		} else {
			temp, err := util.SeedFileName(jobDirectory)
			seedFileName = temp
			if err != nil {
				util.PrintUtil("ERROR: %s\n", err.Error())
				return "", err
			}
		}

		version := objects.SeedFromImageLabel(origImg).SeedVersion
		ValidateSeedFile(false, "", version, seedFileName, common_const.SchemaManifest)
		seed := objects.SeedFromManifestFile(seedFileName)

		util.PrintUtil("INFO: An image with the name %s already exists. ", img)
		// Bump the package patch version
		if pp {
			pkgVersion := strings.Split(seed.Job.PackageVersion, ".")
			patchVersion, _ := strconv.Atoi(pkgVersion[2])
			pkgVersion[2] = strconv.Itoa(patchVersion + 1)
			seed.Job.PackageVersion = strings.Join(pkgVersion, ".")
			util.PrintUtil("The package patch version will be increased to %s.\n",
				seed.Job.PackageVersion)

			// Bump the package minor verion
		} else if pm {
			pkgVersion := strings.Split(seed.Job.PackageVersion, ".")
			minorVersion, _ := strconv.Atoi(pkgVersion[1])
			pkgVersion[1] = strconv.Itoa(minorVersion + 1)
			pkgVersion[2] = "0"
			seed.Job.PackageVersion = strings.Join(pkgVersion, ".")

			util.PrintUtil("The package version will be increased to %s.\n",
				seed.Job.PackageVersion)

			// Bump the package major version
		} else if P {
			pkgVersion := strings.Split(seed.Job.PackageVersion, ".")
			majorVersion, _ := strconv.Atoi(pkgVersion[0])
			pkgVersion[0] = strconv.Itoa(majorVersion + 1)
			pkgVersion[1] = "0"
			pkgVersion[2] = "0"
			seed.Job.PackageVersion = strings.Join(pkgVersion, ".")

			util.PrintUtil("The major package version will be increased to %s.\n",
				seed.Job.PackageVersion)
		}
		// Bump the job patch version
		if jp {
			jobVersion := strings.Split(seed.Job.JobVersion, ".")
			patchVersion, _ := strconv.Atoi(jobVersion[2])
			jobVersion[2] = strconv.Itoa(patchVersion + 1)
			seed.Job.JobVersion = strings.Join(jobVersion, ".")
			util.PrintUtil("The job patch version will be increased to %s.\n",
				seed.Job.JobVersion)

			// Bump the job minor verion
		} else if jm {
			jobVersion := strings.Split(seed.Job.JobVersion, ".")
			minorVersion, _ := strconv.Atoi(jobVersion[1])
			jobVersion[1] = strconv.Itoa(minorVersion + 1)
			jobVersion[2] = "0"
			seed.Job.JobVersion = strings.Join(jobVersion, ".")
			util.PrintUtil("The minor job version will be increased to %s.\n",
				seed.Job.JobVersion)

			// Bump the job major verion
		} else if J {
			jobVersion := strings.Split(seed.Job.JobVersion, ".")
			majorVersion, _ := strconv.Atoi(jobVersion[0])
			jobVersion[0] = strconv.Itoa(majorVersion + 1)
			jobVersion[1] = "0"
			jobVersion[2] = "0"
			seed.Job.JobVersion = strings.Join(jobVersion, ".")

			util.PrintUtil("The major job version will be increased to %s.\n",
				seed.Job.JobVersion)
		}
		if !J && !jm && !jp && !P && !pm && !pp {
			util.PrintUtil("ERROR: No tag deconfliction method specified. Aborting seed publish.\n")
			util.PrintUtil("Exiting seed...\n")
			return "", errors.New("Image exists and no tag deconfliction method specified.")
		}

		img = objects.BuildImageName(&seed)
		util.PrintUtil("\nNew image name: %s\n", img)

		// write version back to the seed manifest
		seedJSON, _ := json.MarshalIndent(&seed, "", "  ")
		err := ioutil.WriteFile(seedFileName, seedJSON, os.ModePerm)
		if err != nil {
			util.PrintUtil("ERROR: Error occurred writing updated seed version to %s.\n%s\n",
				seedFileName, err.Error())
			return "", errors.New("Error updating seed version in manifest.")
		}

		// Build Docker image
		util.PrintUtil("INFO: Building %s\n", img)
		buildArgs := []string{"build", "-t", img, jobDirectory}
		if util.DockerVersionHasLabel() {
			// Set the seed.manifest.json contents as an image label
			label := "com.ngageoint.seed.manifest=" + objects.GetManifestLabel(seedFileName)
			buildArgs = append(buildArgs, "--label", label)
		}
		util.PrintUtil("INFO: Running Docker command\n:docker %s\n", strings.Join(buildArgs, " "))
		rebuildCmd := exec.Command("docker", buildArgs...)
		var errs bytes.Buffer
		if util.StdErr != nil {
			rebuildCmd.Stderr = io.MultiWriter(util.StdErr, &errs)
		} else {
			rebuildCmd.Stderr = &errs
		}
		rebuildCmd.Stdout = util.StdErr

		// Run docker build
		rebuildCmd.Run()

		// check for errors on stderr
		if errs.String() != "" {
			util.PrintUtil("ERROR: Error re-building image '%s':\n%s\n",
				img, errs.String())
			util.PrintUtil("Exiting seed...\n")
			return "", errors.New(errs.String())
		}

		// Set final image name to tag + image
		origImg = img
		img = tag + img
	}

	err := util.Tag(origImg, img)
	if err != nil {
		return img, err
	}

	err = util.Push(img)
	if err != nil {
		return img, err
	}

	err = util.RemoveImage(img)
	if err != nil {
		return img, err
	}

	return img, nil
}

//PrintPublishUsage prints the seed publish usage information, then exits the program
func PrintPublishUsage() {
	util.PrintUtil("\nUsage:\tseed publish [-in IMAGE_NAME] [-M MANIFEST] [-r REGISTRY_NAME] [-O ORG_NAME] [-u username] [-p password] [Conflict Options]\n")
	util.PrintUtil("\nAllows for the publish of seed compliant images.\n")
	util.PrintUtil("\nOptions:\n")
	util.PrintUtil("  -%s -%s Docker image name to publish\n",
		constants.ShortImgNameFlag, constants.ImgNameFlag)
	util.PrintUtil("  -%s -%s\t  Manifest file to use if an image name is not specified (default is seed.manifest.json within the current directory).\n",
		constants.ShortManifestFlag, constants.ManifestFlag)
	util.PrintUtil("  -%s  -%s\t Specifies a specific registry to publish the image\n",
		constants.ShortRegistryFlag, constants.RegistryFlag)
	util.PrintUtil("  -%s  -%s\t Specifies a specific organization to publish the image\n",
		constants.ShortOrgFlag, constants.OrgFlag)
	util.PrintUtil("  -%s  -%s\t Username to login if needed to publish images (default anonymous).\n",
		constants.ShortUserFlag, constants.UserFlag)
	util.PrintUtil("  -%s  -%s\t Password to login if needed to publish images (default anonymous).\n",
		constants.ShortPassFlag, constants.PassFlag)
	util.PrintUtil("  -%s\t\t Overwrite remote image if publish conflict found\n",
		constants.ForcePublishFlag)

	util.PrintUtil("\nConflict Options:\n")
	util.PrintUtil("If the force flag (-f) is not set, the following options specify how a publish conflict is handled:\n")
	util.PrintUtil("  -%s -%s Specifies the directory containing the seed.manifest.json and dockerfile to rebuild the image.\n",
		constants.ShortJobDirectoryFlag, constants.JobDirectoryFlag)
	util.PrintUtil("  -%s\t\tForce Patch version bump of 'packageVersion' in manifest on disk if publish conflict found\n",
		constants.PkgVersionPatch)
	util.PrintUtil("  -%s\t\tForce Minor version bump of 'packageVersion' in manifest on disk if publish conflict found\n",
		constants.PkgVersionMinor)
	util.PrintUtil("  -%s\t\tForce Major version bump of 'packageVersion' in manifest on disk if publish conflict found\n",
		constants.PkgVersionMajor)
	util.PrintUtil("  -%s\t\tForce Patch version bump of 'jobVersion' in manifest on disk if publish conflict found\n",
		constants.JobVersionPatch)
	util.PrintUtil("  -%s\t\tForce Minor version bump of 'jobVersion' in manifest on disk if publish conflict found\n",
		constants.JobVersionMinor)
	util.PrintUtil("  -%s\t\tForce Major version bump of 'jobVersion' in manifest on disk if publish conflict found\n",
		constants.JobVersionMajor)

	util.PrintUtil("\nExample: \tseed publish -in example-0.1.3-seed:0.1.3 -r my.registry.address -jm -P\n")
	util.PrintUtil("\nIf example-0.1.3-seed:0.1.3 does not exist on the registry the image will be published there.")
	util.PrintUtil("If it does exist this will build a new image example-0.2.0-seed:1.0.0 and publish it to the registry\n")
	return
}
