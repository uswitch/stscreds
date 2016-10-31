package main

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"time"
)

type ReadCommand struct {
	Key     string
	Expiry  time.Duration
	Profile string
}

func (cmd *ReadCommand) Execute() error {
	err := cmd.ensureCredentialsFresh()
	if err != nil {
		return fmt.Errorf("error checking credentials: %s", err.Error())
	}

	path, err := defaultAWSCredentialsPath()
	cfg, err := ini.Load(path)
	if err != nil {
		return fmt.Errorf("error reading credentials: %s", err.Error())
	}

	section, err := cfg.GetSection(cmd.Profile)
	if err != nil {
		return fmt.Errorf("couldn't read [%s] section: %s", cmd.Profile, err.Error())
	}

	if !section.HasKey(cmd.Key) {
		return fmt.Errorf("%s not found in [%s]", cmd.Key, cmd.Profile)
	}
	value := section.Key(cmd.Key).String()

	fmt.Print(value)

	return nil
}

func (cmd *ReadCommand) isCredentialsFresh() (bool, error) {
	ex, err := readExpiry()
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	expired := time.Now().After(ex.Expiry)
	return !expired, nil
}

func (cmd *ReadCommand) ensureCredentialsFresh() error {
	fresh, err := cmd.isCredentialsFresh()
	if err != nil {
		return err
	}
	if !fresh {
		return cmd.refreshCredentials()
	}

	return nil
}

func (cmd *ReadCommand) refreshCredentials() error {
	fmt.Println("Credentials have expired, need to refresh.")
	auth := &AuthCommand{Expiry: *expires}
	return auth.Execute()
}
