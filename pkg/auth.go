package stscreds

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type AuthCommand struct {
	Expiry              time.Duration
	OutputAsEnvVariable bool
	Profile             string
}

func askUserForToken() (string, error) {
	fmt.Fprintf(os.Stderr, "Please enter MFA token: ")

	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading token: %s", err.Error())
	}
	return strings.Trim(text, " \r\n"), nil
}

func (cmd *AuthCommand) Execute() error {
	err := ensureAwsDir()
	if err != nil {
		return fmt.Errorf("Error ensuring .aws directory: %s", err)
	}

	limitedCreds, err := DefaultLimitedAccessCredentials(cmd.Profile)
	if err != nil {
		return err
	}

	limitedAccessSession, err := limitedCreds.NewSession()
	if err != nil {
		return err
	}
	username, err := currentUserName(limitedAccessSession)
	if err != nil {
		return fmt.Errorf("couldn't request current user: %s\n", err.Error())
	}

	fmt.Fprintf(os.Stderr, "Current user: %s. ", username)

	token, err := askUserForToken()
	if err != nil {
		return fmt.Errorf("error requesting mfa token: %s", err.Error())
	}

	generatedCredentials, err := requestNewSTSToken(limitedAccessSession, username, token, cmd.Expiry, cmd.Profile)
	if err != nil {
		return fmt.Errorf("error requesting credentials: %s", err.Error())
	}

	tc, err := DefaultTemporaryCredentials(cmd.Profile)
	if err != nil {
		return err
	}
	tc.UpdateCredentials(generatedCredentials)
	err = tc.Save()
	if err != nil {
		return err
	}

	err = limitedCreds.RecordExpiry(generatedCredentials.Expiry)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Wrote credentials to %s\n", tc.path)

	if cmd.OutputAsEnvVariable {
		envVarExportsOutput(generatedCredentials)
	}

	return nil
}
