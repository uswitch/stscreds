package main

import (
	"bufio"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"os"
	"strings"
	"time"
)

type AuthCommand struct {
	Expiry              time.Duration
	OutputAsEnvVariable bool
	Profile             string
}

func askUserForToken(sess *session.Session) (string, error) {
	fmt.Fprintf(os.Stderr, "Please enter MFA token: ")

	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading token: %s", err.Error())
	}
	return strings.Trim(text, " \n"), nil
}

func writeSTSCredentials(creds *Credentials, path, profile string) error {
	w := newCredentialsFileWriter(path)

	err := w.Output(creds, profile)
	if err != nil {
		return err
	}
	return nil
}

func (cmd *AuthCommand) Execute() error {
	path, err := defaultAWSCredentialsPath()
	if err != nil {
		return fmt.Errorf("error determining aws credentials path: %s", err.Error())
	}

	sess, err := newLimitedAccessSession(cmd.Profile)
	if err != nil {
		return err
	}
	username, err := currentUserName(sess)
	if err != nil {
		return fmt.Errorf("couldn't request current user: %s\n", err.Error())
	}

	fmt.Fprintf(os.Stderr, "Current user: %s. ", username)

	token, err := askUserForToken(sess)
	if err != nil {
		return fmt.Errorf("error requesting mfa token: %s", err.Error())
	}

	creds, err := requestNewSTSToken(sess, username, token, cmd.Expiry, cmd.Profile)
	if err != nil {
		return fmt.Errorf("error requesting credentials: %s", err.Error())
	}

	err = writeSTSCredentials(creds, path, cmd.Profile)
	if err != nil {
		return fmt.Errorf("error writing credentials %s: %s", path, err.Error())
	}
	err = writeExpiry(creds)
	if err != nil {
		return fmt.Errorf("error writing credentials expiry: %s", err.Error())
	}

	fmt.Fprintf(os.Stderr, "Wrote credentials to %s\n", path)

	if cmd.OutputAsEnvVariable {
		envVarExportsOutput(creds)
	}

	return nil
}
