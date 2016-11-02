package main

import (
	"bufio"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"gopkg.in/ini.v1"
	"os"
	"strings"
)

type InitCommand struct {
	Profile string
}

func (c *InitCommand) credentialsFile(path string) (*ini.File, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return ini.Empty(), nil
	}
	if err != nil {
		return nil, err
	}

	return ini.Load(path)
}

func (c *InitCommand) writeFile(accessKey, secretKey, path string) error {
	cfg, err := c.credentialsFile(path)
	if err != nil {
		return err
	}

	sec, _ := cfg.NewSection(c.Profile)
	sec.NewKey("aws_access_key_id", accessKey)
	sec.NewKey("aws_secret_access_key", secretKey)

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)

	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	cfg.WriteTo(w)
	w.Flush()

	return nil
}

func warnOnEnvironmentVariables() {
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		fmt.Fprintf(os.Stderr, "warning: AWS_ACCESS_KEY_ID environment variable set, may override sts credentials initialised in ~/.aws/credentials.\nwarning: AWS_ACCESS_KEY_ID should probably be removed from your environment; check ~/.bash_profile etc.\n")
	}

	if os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		fmt.Fprintf(os.Stderr, "warning: AWS_SECRET_ACCESS_KEY environment variable set, may override sts credentials initialised in ~/.aws/credentials.\nwarning: AWS_SECRET_ACCESS_KEY should probably be removed from your environment; check ~/.bash_profile etc.\n")
	}
}

type Keys struct {
	AccessKey string
	SecretKey string
}

func (k *Keys) Valid() (bool, error) {
	sess := session.New(&aws.Config{Credentials: credentials.NewStaticCredentials(k.AccessKey, k.SecretKey, "")})
	_, err := getUser(sess)
	if err != nil {
		return false, err
	}

	return true, nil
}

func readFromPrompt() (*Keys, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stderr, "AWS Access Key: ")
	text, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	accessKey := strings.Trim(text, " \n")
	fmt.Fprintf(os.Stderr, "AWS Secret Access Key: ")
	text, err = reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	secretKey := strings.Trim(text, " \n")

	return &Keys{accessKey, secretKey}, nil
}

func readAWSKeys() (*Keys, error) {
	keys, err := readFromPrompt()
	if err != nil {
		return nil, err
	}

	_, err = keys.Valid()
	if err != nil {
		return nil, err
	}

	return keys, err
}

func (cmd *InitCommand) Execute() error {
	warnOnEnvironmentVariables()

	creds, err := DefaultLimitedAccessCredentials(cmd.Profile)
	if err != nil {
		return err
	}

	keys, err := readAWSKeys()
	if err != nil {
		return fmt.Errorf("error with aws credentials: %s", err.Error())
	}

	err = creds.Initialise(keys)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Successfully wrote %s\n", creds.path)

	// path, err := limitedAccessCredentialsPath()
	// if err != nil {
	// 	return err
	// }
	// err = os.MkdirAll(filepath.Dir(path), 0700)
	// if err != nil {
	// 	return err
	// }
	//
	// keys, err := readAWSKeys()
	// if err != nil {
	// 	return fmt.Errorf("error with aws credentials: %s", err.Error())
	// }
	//
	// err = cmd.writeFile(keys.AccessKey, keys.SecretKey, path)
	// if err != nil {
	// 	return err
	// }
	//
	// fmt.Fprintf(os.Stderr, "Successfully wrote %s\n", path)
	return nil
}
