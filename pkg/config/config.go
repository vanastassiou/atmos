package config

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"

	u "github.com/cloudposse/atmos/pkg/utils"
)

var NotFound = errors.New("\n'atmos.yaml' CLI config files not found in any of the searched paths: system dir, home dir, current dir, ENV vars." +
	"\nYou can download a sample config and adapt it to your requirements from " +
	"https://raw.githubusercontent.com/cloudposse/atmos/master/examples/complete/atmos.yaml")

// InitCliConfig finds and merges CLI configurations in the following order: system dir, home dir, current dir, ENV vars, command-line arguments
// https://dev.to/techschoolguru/load-config-from-file-environment-variables-in-golang-with-viper-2j2d
// https://medium.com/@bnprashanth256/reading-configuration-files-and-environment-variables-in-go-golang-c2607f912b63
func InitCliConfig(configAndStacksInfo ConfigAndStacksInfo, processStacks bool) (CliConfiguration, error) {
	// cliConfig is loaded from the following locations (from lower to higher priority):
	// system dir (`/usr/local/etc/atmos` on Linux, `%LOCALAPPDATA%/atmos` on Windows)
	// home dir (~/.atmos)
	// current directory
	// ENV vars
	// Command-line arguments

	var cliConfig CliConfiguration
	var err error
	var verbose = processStacks

	// Check `ATMOS_LOGS_VERBOSE` ENV var
	// If it's set to `true`, log verbose even during the CLI config initialization
	logVerboseEnvVar := false
	logVerboseEnvVarFound := false
	logVerboseEnvVarStr := os.Getenv("ATMOS_LOGS_VERBOSE")
	if len(logVerboseEnvVarStr) > 0 {
		u.PrintInfoVerbose(verbose, fmt.Sprintf("Found ENV var ATMOS_LOGS_VERBOSE=%s", logVerboseEnvVarStr))
		logVerboseEnvVar, err = strconv.ParseBool(logVerboseEnvVarStr)
		if err != nil {
			return cliConfig, err
		}
		logVerboseEnvVarFound = true
	}

	var printVerbose = verbose && logVerboseEnvVarFound && logVerboseEnvVar

	if printVerbose {
		u.PrintInfo("\nSearching, processing and merging atmos CLI configurations (atmos.yaml) in the following order:")
		fmt.Println("system dir, home dir, current dir, ENV vars, command-line arguments")
		fmt.Println()
	}

	configFound := false
	var found bool

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetTypeByDefaultValue(true)

	// Default configuration values
	v.SetDefault("components.helmfile.use_eks", true)

	// Process config in system folder
	configFilePath1 := ""

	// https://pureinfotech.com/list-environment-variables-windows-10/
	// https://docs.microsoft.com/en-us/windows/deployment/usmt/usmt-recognized-environment-variables
	// https://softwareengineering.stackexchange.com/questions/299869/where-is-the-appropriate-place-to-put-application-configuration-files-for-each-p
	// https://stackoverflow.com/questions/37946282/why-does-appdata-in-windows-7-seemingly-points-to-wrong-folder
	if runtime.GOOS == "windows" {
		appDataDir := os.Getenv(WindowsAppDataEnvVar)
		if len(appDataDir) > 0 {
			configFilePath1 = appDataDir
		}
	} else {
		configFilePath1 = SystemDirConfigFilePath
	}

	if len(configFilePath1) > 0 {
		configFile1 := path.Join(configFilePath1, CliConfigFileName)
		found, err = processConfigFile(printVerbose, configFile1, v)
		if err != nil {
			return cliConfig, err
		}
		if found {
			configFound = true
		}
	}

	// Process config in user's HOME dir
	configFilePath2, err := homedir.Dir()
	if err != nil {
		return cliConfig, err
	}
	configFile2 := path.Join(configFilePath2, ".atmos", CliConfigFileName)
	found, err = processConfigFile(printVerbose, configFile2, v)
	if err != nil {
		return cliConfig, err
	}
	if found {
		configFound = true
	}

	// Process config in the current dir
	configFilePath3, err := os.Getwd()
	if err != nil {
		return cliConfig, err
	}
	configFile3 := path.Join(configFilePath3, CliConfigFileName)
	found, err = processConfigFile(printVerbose, configFile3, v)
	if err != nil {
		return cliConfig, err
	}
	if found {
		configFound = true
	}

	// Process config from the path in ENV var `ATMOS_CLI_CONFIG_PATH`
	configFilePath4 := os.Getenv("ATMOS_CLI_CONFIG_PATH")
	if len(configFilePath4) > 0 {
		u.PrintInfoVerbose(printVerbose, fmt.Sprintf("Found ENV var ATMOS_CLI_CONFIG_PATH=%s", configFilePath4))
		configFile4 := path.Join(configFilePath4, CliConfigFileName)
		found, err = processConfigFile(printVerbose, configFile4, v)
		if err != nil {
			return cliConfig, err
		}
		if found {
			configFound = true
		}
	}

	// Process config from the path specified in the Terraform provider (which calls into the atmos code)
	if configAndStacksInfo.AtmosCliConfigPath != "" {
		configFilePath5 := configAndStacksInfo.AtmosCliConfigPath
		if len(configFilePath5) > 0 {
			configFile5 := path.Join(configFilePath5, CliConfigFileName)
			found, err = processConfigFile(printVerbose, configFile5, v)
			if err != nil {
				return cliConfig, err
			}
			if found {
				configFound = true
			}
		}
	}

	if !configFound {
		return cliConfig, NotFound
	}

	// https://gist.github.com/chazcheadle/45bf85b793dea2b71bd05ebaa3c28644
	// https://sagikazarmark.hu/blog/decoding-custom-formats-with-viper/
	err = v.Unmarshal(&cliConfig)
	if err != nil {
		return cliConfig, err
	}

	// Set log verbose for the command that is being executed after the CLI config gets processed
	// `logs.verbose` can be set in `atmos.yaml` or overridden by `ATMOS_LOGS_VERBOSE` ENV var
	if logVerboseEnvVarFound {
		cliConfig.Logs.Verbose = logVerboseEnvVar
	}

	// Process ENV vars
	err = processEnvVars(&cliConfig)
	if err != nil {
		return cliConfig, err
	}

	// Process command-line args
	err = processCommandLineArgs(&cliConfig, configAndStacksInfo)
	if err != nil {
		return cliConfig, err
	}

	// Process the base path specified in the Terraform provider (which calls into the atmos code)
	// This overrides all other atmos base path configs (`atmos.yaml`, ENV var `ATMOS_BASE_PATH`)
	if configAndStacksInfo.AtmosBasePath != "" {
		cliConfig.BasePath = configAndStacksInfo.AtmosBasePath
	}

	// Check config
	err = checkConfig(cliConfig)
	if err != nil {
		return cliConfig, err
	}

	// Convert stacks base path to absolute path
	stacksBasePath := path.Join(cliConfig.BasePath, cliConfig.Stacks.BasePath)
	stacksBaseAbsPath, err := filepath.Abs(stacksBasePath)
	if err != nil {
		return cliConfig, err
	}
	cliConfig.StacksBaseAbsolutePath = stacksBaseAbsPath

	// Convert the included stack paths to absolute paths
	includeStackAbsPaths, err := u.JoinAbsolutePathWithPaths(stacksBaseAbsPath, cliConfig.Stacks.IncludedPaths)
	if err != nil {
		return cliConfig, err
	}
	cliConfig.IncludeStackAbsolutePaths = includeStackAbsPaths

	// Convert the excluded stack paths to absolute paths
	excludeStackAbsPaths, err := u.JoinAbsolutePathWithPaths(stacksBaseAbsPath, cliConfig.Stacks.ExcludedPaths)
	if err != nil {
		return cliConfig, err
	}
	cliConfig.ExcludeStackAbsolutePaths = excludeStackAbsPaths

	// Convert terraform dir to absolute path
	terraformBasePath := path.Join(cliConfig.BasePath, cliConfig.Components.Terraform.BasePath)
	terraformDirAbsPath, err := filepath.Abs(terraformBasePath)
	if err != nil {
		return cliConfig, err
	}
	cliConfig.TerraformDirAbsolutePath = terraformDirAbsPath

	// Convert helmfile dir to absolute path
	helmfileBasePath := path.Join(cliConfig.BasePath, cliConfig.Components.Helmfile.BasePath)
	helmfileDirAbsPath, err := filepath.Abs(helmfileBasePath)
	if err != nil {
		return cliConfig, err
	}
	cliConfig.HelmfileDirAbsolutePath = helmfileDirAbsPath

	if processStacks {
		// If the specified stack name is a logical name, find all stack config files in the provided paths
		stackConfigFilesAbsolutePaths, stackConfigFilesRelativePaths, stackIsPhysicalPath, err := FindAllStackConfigsInPathsForStack(
			cliConfig,
			configAndStacksInfo.Stack,
			includeStackAbsPaths,
			excludeStackAbsPaths,
		)

		if err != nil {
			return cliConfig, err
		}

		if len(stackConfigFilesAbsolutePaths) < 1 {
			j, err := yaml.Marshal(includeStackAbsPaths)
			if err != nil {
				return cliConfig, err
			}
			errorMessage := fmt.Sprintf("\nNo stack config files found in the provided "+
				"paths:\n%s\n\nCheck if `base_path`, 'stacks.base_path', 'stacks.included_paths' and 'stacks.excluded_paths' are correctly set in CLI config "+
				"files or ENV vars.", j)
			return cliConfig, errors.New(errorMessage)
		}

		cliConfig.StackConfigFilesAbsolutePaths = stackConfigFilesAbsolutePaths
		cliConfig.StackConfigFilesRelativePaths = stackConfigFilesRelativePaths

		if stackIsPhysicalPath {
			u.PrintInfoVerbose(printVerbose, fmt.Sprintf("\nThe stack '%s' matches the stack config file %s\n",
				configAndStacksInfo.Stack,
				stackConfigFilesRelativePaths[0]),
			)
			cliConfig.StackType = "Directory"
		} else {
			// The stack is a logical name
			cliConfig.StackType = "Logical"
		}
	}

	if printVerbose {
		u.PrintInfo("\nFinal CLI configuration:")
		err = u.PrintAsYAML(cliConfig)
		if err != nil {
			return cliConfig, err
		}
	}

	cliConfig.Initialized = true
	return cliConfig, nil
}

// https://github.com/NCAR/go-figure
// https://github.com/spf13/viper/issues/181
// https://medium.com/@bnprashanth256/reading-configuration-files-and-environment-variables-in-go-golang-c2607f912b63
func processConfigFile(verbose bool, path string, v *viper.Viper) (bool, error) {
	if !u.FileExists(path) {
		u.PrintInfoVerbose(verbose, fmt.Sprintf("No config file 'atmos.yaml' found in path '%s'.", path))
		return false, nil
	}

	u.PrintInfoVerbose(verbose, fmt.Sprintf("Found CLI config in '%s'", path))

	reader, err := os.Open(path)
	if err != nil {
		return false, err
	}

	defer func(reader *os.File) {
		err := reader.Close()
		if err != nil {
			u.PrintError(fmt.Errorf("error closing file '" + path + "'. " + err.Error()))
		}
	}(reader)

	err = v.MergeConfig(reader)
	if err != nil {
		return false, err
	}

	u.PrintInfoVerbose(verbose, fmt.Sprintf("Processed CLI config '%s'", path))

	return true, nil
}
