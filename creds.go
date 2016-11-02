package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"gopkg.in/ini.v1"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

func homePath(paths ...string) (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	parts := append([]string{u.HomeDir}, paths...)
	return filepath.Join(parts...), nil
}

// credentials managed by stscreds that are requested through STS
type TemporaryCredentials struct {
	profile string
	path    string

	latestCredentials *Credentials
}

func (c *TemporaryCredentials) UpdateCredentials(credentials *Credentials) {
	c.latestCredentials = credentials
}

func (c *TemporaryCredentials) Save() error {
	cfg, err := ini.LooseLoad(c.path)
	if err != nil {
		return err
	}

	sec, err := cfg.NewSection(c.profile)
	if err != nil {
		return err
	}

	_, err = sec.NewKey("aws_access_key_id", c.latestCredentials.AccessKey)
	if err != nil {
		return err
	}
	_, err = sec.NewKey("aws_secret_access_key", c.latestCredentials.SecretKey)
	if err != nil {
		return err
	}
	_, err = sec.NewKey("aws_session_token", c.latestCredentials.SessionToken)
	if err != nil {
		return err
	}

	return cfg.SaveTo(c.path)
}

func (c *TemporaryCredentials) Read(key string) (interface{}, error) {
	cfg, err := ini.Load(c.path)
	if err != nil {
		return nil, err
	}
	sec, err := cfg.GetSection(c.profile)
	if err != nil {
		return nil, err
	}

	if !sec.HasKey(key) {
		return nil, fmt.Errorf("key %s not found", key)
	}

	return sec.Key(key).String(), nil
}

func DefaultTemporaryCredentials(profile string) (*TemporaryCredentials, error) {
	path, err := homePath(".aws", "credentials")
	if err != nil {
		return nil, err
	}
	return &TemporaryCredentials{profile: profile, path: path}, nil
}

// these are the credentials used by the program to request credentials through STS
// and retrieve user info through IAM etc.
type LimitedAccessCredentials struct {
	path    string
	profile string
}

const ExpiresKey = "temp_credentials_expire"

func (c *LimitedAccessCredentials) IsTemporaryCredentialsExpired(now time.Time) (bool, error) {
	cfg, err := c.file()
	if err != nil {
		return false, err
	}

	sec, err := cfg.NewSection(c.profile)
	if err != nil {
		return false, err
	}

	if !sec.HasKey(ExpiresKey) {
		return false, nil
	}

	k, err := sec.GetKey(ExpiresKey)
	if err != nil {
		return false, err
	}

	expires, err := k.Time()
	if err != nil {
		return false, err
	}

	return now.After(expires), nil
}

func (c *LimitedAccessCredentials) RecordExpiry(expiresAt time.Time) error {
	cfg, err := c.file()
	if err != nil {
		return err
	}

	sec, err := cfg.NewSection(c.profile)
	if err != nil {
		return err
	}

	_, err = sec.NewKey(ExpiresKey, expiresAt.Format(time.RFC3339))
	if err != nil {
		return err
	}

	return cfg.SaveTo(c.path)
}

func (c *LimitedAccessCredentials) Exist() (bool, error) {
	fi, err := os.Stat(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("unexpected error with stscreds file: %s", err.Error())
	}
	if !fi.Mode().IsRegular() {
		return false, fmt.Errorf("%s is not a regular file", c.path)
	}
	return true, nil
}

func DefaultLimitedAccessCredentials(profile string) (*LimitedAccessCredentials, error) {
	filepath, err := homePath(".stscreds", "credentials")
	if err != nil {
		return nil, err
	}
	return &LimitedAccessCredentials{path: filepath, profile: profile}, nil
}

func (c *LimitedAccessCredentials) file() (*ini.File, error) {
	return ini.LooseLoad(c.path)
}

func (c *LimitedAccessCredentials) Initialise(keys *Keys) error {
	err := os.MkdirAll(filepath.Dir(c.path), 0700)
	if err != nil {
		return err
	}

	cfg, err := c.file()
	if err != nil {
		return err
	}

	sec, err := cfg.NewSection(c.profile)
	if err != nil {
		return err
	}

	_, err = sec.NewKey("aws_access_key_id", keys.AccessKey)
	if err != nil {
		return err
	}
	_, err = sec.NewKey("aws_secret_access_key", keys.SecretKey)
	if err != nil {
		return err
	}

	return nil
}

func (c *LimitedAccessCredentials) NewSession() (*session.Session, error) {
	stsCreds := credentials.NewSharedCredentials(c.path, c.profile)
	return session.New(&aws.Config{Credentials: stsCreds}), nil
}
