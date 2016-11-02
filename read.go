package main

import (
	"fmt"
	"time"
)

type ReadCommand struct {
	Key     string
	Expiry  time.Duration
	Profile string
}

type ExpiredCredentialsErr string

func (e ExpiredCredentialsErr) Error() string {
	return "Credentials have expired"
}

func (cmd *ReadCommand) Execute() error {
	limitedCredentials, err := DefaultLimitedAccessCredentials(cmd.Profile)
	if err != nil {
		return err
	}

	expired, err := limitedCredentials.IsTemporaryCredentialsExpired(time.Now())
	if err != nil {
		return err
	}

	if expired {
		return ExpiredCredentialsErr(cmd.Profile)
	}

	creds, err := DefaultTemporaryCredentials(cmd.Profile)
	if err != nil {
		return err
	}

	value, err := creds.Read(cmd.Key)
	if err != nil {
		return fmt.Errorf("error reading %s: %s", cmd.Key, err.Error())
	}

	fmt.Print(value)

	return nil
}
