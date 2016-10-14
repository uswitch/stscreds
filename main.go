package main

import (
	"fmt"
	"github.com/alecthomas/kingpin"
	"os"
	"os/user"
	"path/filepath"
)

var (
	initCommand = kingpin.Command("init", "Initialise stscreds. Creates ~/.stscreds/credentials.")
	expires     = kingpin.Flag("expires", "Credentials expiry").Default("12h").Duration()

	authCommand    = kingpin.Command("auth", "Authenticates with AWS and requests a temporary session token.")
	envVarTemplate = authCommand.Flag("output-env", "Additionally write environment variable exports to stdout.").Bool()

	readCommand = kingpin.Command("read", "Read keys from ~/.aws/credentials and print to stdout.")
	readKey     = readCommand.Arg("key", "Key to read from credentials file: aws_access_key_id, aws_secret_access_key, aws_session_token.").String()

	userCommand = kingpin.Command("whoami", "Print details about current user.")
)

var versionNumber string

func versionString() string {
	if versionNumber != "" {
		return versionNumber
	}
	return "DEVELOPMENT"
}

func homePath(paths ...string) (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	parts := append([]string{u.HomeDir}, paths...)
	return filepath.Join(parts...), nil
}

type Command interface {
	Execute() error
}

func cmdFailWithoutInitialisation(cmd Command) error {
	exist, err := credentialsExist()
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("no credentials found, please run init first.")
	}
	return cmd.Execute()
}

func handle() error {
	switch kingpin.Parse() {
	case "init":
		cmd := &InitCommand{}
		return cmd.Execute()
	case "whoami":
		return cmdFailWithoutInitialisation(&WhoAmI{})
	case "auth":
		return cmdFailWithoutInitialisation(&AuthCommand{Expiry: *expires, OutputAsEnvVariable: *envVarTemplate})
	case "read":
		return cmdFailWithoutInitialisation(&ReadCommand{Key: *readKey, Expiry: *expires})
	}
	return nil
}

func main() {
	kingpin.Version(versionString())
	err := handle()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(2)
	}
}
