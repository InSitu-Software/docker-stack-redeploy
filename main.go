package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"strconv"
	"time"

	"os/exec"

	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/schema"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

func init() {
	pflag.StringP("compose-file", "c", "docker-compose.yml", "Path to a Compose file")
	pflag.Bool("with-registry-auth", false, "Send registry authentication details to Swarm agents")

	pflag.StringP("docker-binary", "d", "docker", "Alternative docker binary")
	pflag.StringP("prefix", "p", "", "Prefix to be used for config prefix")
	pflag.StringP("stack", "s", "", "Stack to be redployed")
	pflag.StringP("workdir", "w", ".", "Specify workdir")
	pflag.BoolP("output", "o", false, "Output YAML rather than redeploy")

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	if viper.GetString("stack") == "" {
		logrus.Fatal("Missing stack name")
	}
}

func main() {

	prefix := getPrefix()

	config := parseComposefile(viper.GetString("compose-file"))

	// Prefix toplevel configs
	configMap := make(map[string]string)
	newConfig := make(map[string]composetypes.ConfigObjConfig)

	for oldKey, data := range config.Configs {
		newKey := prefix + oldKey
		configMap[oldKey] = newKey
		newConfig[newKey] = data
	}
	config.Configs = newConfig

	// prefix service configs
	newServices := composetypes.Services{}
	for _, serviceConfig := range config.Services {
		var newServiceConfigConfig []composetypes.ServiceConfigObjConfig
		for _, configData := range serviceConfig.Configs {
			configData.Source = configMap[configData.Source]
			newServiceConfigConfig = append(newServiceConfigConfig, configData)
		}
		serviceConfig.Configs = newServiceConfigConfig
		newServices = append(newServices, serviceConfig)
	}
	config.Services = newServices

	b, _ := yaml.Marshal(config)

	if viper.GetBool("output") {
		fmt.Println(string(b))
		os.Exit(0)
	}

	// call docker deploy
	f, err := ioutil.TempFile(viper.GetString("workdir"), prefix)
	errorCheck(err)

	if _, err := f.Write(b); err != nil {
		errorCheck(err)
	}
	f.Close()

	args := []string{"stack", "deploy", "-c"}
	args = append(
		args,
		f.Name(),
	)

	if viper.GetBool("with-registry-auth") {
		args = append(args, "--with-registry-auth")
	}

	args = append(args, viper.GetString("stack"))

	cmd := exec.Command(viper.GetString("docker-binary"), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logrus.WithError(err).Error("Running docker failed")
	}

	errorCheck(os.Remove(viper.GetString("workdir") + "/" + f.Name()))
}

func parseComposefile(composeFile string) *composetypes.Config {
	fileBytes, err := ioutil.ReadFile(composeFile)
	errorCheck(err)

	parsedConfig, err := loader.ParseYAML(fileBytes)
	errorCheck(err)

	data, err := composetypes.ConfigFile{
		Filename: composeFile,
		Config:   parsedConfig,
	}, nil
	errorCheck(err)

	var details composetypes.ConfigDetails
	details.WorkingDir = "."
	details.ConfigFiles = []composetypes.ConfigFile{data}
	details.Version = schema.Version(details.ConfigFiles[0].Config)
	details.Environment, err = buildEnvironment(os.Environ())

	config, err := loader.Load(details)
	errorCheck(err)

	return config
}

func buildEnvironment(env []string) (map[string]string, error) {
	result := make(map[string]string, len(env))
	for _, s := range env {
		// if value is empty, s is like "K=", not "K".
		if !strings.Contains(s, "=") {
			return result, errors.Errorf("unexpected environment %q", s)
		}
		kv := strings.SplitN(s, "=", 2)
		result[kv[0]] = kv[1]
	}
	return result, nil
}

func getPrefix() string {
	// determine prefix
	prefix := viper.GetString("prefix")
	if prefix == "" {
		prefix = strconv.Itoa(
			int(
				time.Now().Unix(),
			),
		)
	}

	prefix = viper.GetString("stack") + "_" + prefix + "_"

	return prefix
}

func errorCheck(err error) {
	if err != nil {
		logrus.WithError(err).Fatal()
	}
}
