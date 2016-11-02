package main

import (
	"fmt"
	"github.com/alecthomas/kingpin"
	"os"
)

var (
	initCommand = kingpin.Command("init", "Initialise stscreds. Creates ~/.stscreds/credentials.")
	expires     = kingpin.Flag("expires", "Credentials expiry").Default("12h").Duration()
	profile     = kingpin.Flag("profile", "AWS profile to manage credentials for.").Default("default").String()

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

type Command interface {
	Execute() error
}

func newCommand(command string) (Command, error) {
	switch command {
	case "whoami":
		return &WhoAmI{Profile: *profile}, nil
	case "auth":
		return &AuthCommand{Expiry: *expires, OutputAsEnvVariable: *envVarTemplate, Profile: *profile}, nil
	case "read":
		return &ReadCommand{Key: *readKey, Expiry: *expires, Profile: *profile}, nil
	}
	return nil, fmt.Errorf("Command not found: %s", command)
}

func handle(command string) error {
	if command == "init" {
		cmd := &InitCommand{Profile: *profile}
		return cmd.Execute()
	}

	creds, err := DefaultLimitedAccessCredentials(*profile)
	if err != nil {
		return err
	}

	exist, err := creds.Exist()
	if err != nil {
		return err
	}

	if !exist {
		return fmt.Errorf("Limited access credentials not found, please run init first.")
	}

	cmd, err := newCommand(command)
	if err != nil {
		return err
	}

	err = cmd.Execute()
	if err == nil {
		return nil
	}

	if _, ok := err.(ExpiredCredentialsErr); ok {
		err = handle("auth")
		if err != nil {
			return err
		}
		return handle(command)
	}

	return err
}

func fatal(e error) {
	fmt.Fprintf(os.Stderr, "error: %s\n", e.Error())
	os.Exit(2)
}

func main() {
	kingpin.Version(versionString())
	command := kingpin.Parse()

	err := handle(command)

	if err != nil {
		fatal(err)
	}
}
