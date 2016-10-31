package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"gopkg.in/ini.v1"
	"os"
	"time"
)

func limitedAccessCredentialsPath() (string, error) {
	return homePath(".stscreds", "credentials")
}

func defaultAWSCredentialsPath() (string, error) {
	return homePath(".aws", "credentials")
}

func expiryPath() (string, error) {
	return homePath(".stscreds", "expiry")
}

func credentialsExist() (bool, error) {
	path, err := limitedAccessCredentialsPath()
	if err != nil {
		return false, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("unexpected error with stscreds file: %s", err.Error())
	}
	if !fi.Mode().IsRegular() {
		return false, fmt.Errorf("credentials path is not a regular file: %s", path)
	}
	return true, nil
}

func newLimitedAccessSession(profile string) (*session.Session, error) {
	p, err := limitedAccessCredentialsPath()
	if err != nil {
		return nil, err
	}
	stsCreds := credentials.NewSharedCredentials(p, "default")
	return session.New(&aws.Config{Credentials: stsCreds}), nil
}

func mfaDevices(sess *session.Session, username string) ([]*iam.MFADevice, error) {
	svc := iam.New(sess)
	resp, err := svc.ListMFADevices(&iam.ListMFADevicesInput{UserName: aws.String(username)})
	if err != nil {
		return nil, err
	}

	return resp.MFADevices, nil
}

func getUser(sess *session.Session) (*iam.User, error) {
	svc := iam.New(sess)
	resp, err := svc.GetUser(&iam.GetUserInput{})

	if err != nil {
		return nil, err
	}

	return resp.User, nil
}

func currentUserName(sess *session.Session) (string, error) {
	user, err := getUser(sess)
	if err != nil {
		return "", err
	}
	return *user.UserName, nil
}

func mfaSerialNumber(sess *session.Session, username string) (string, error) {
	devices, err := mfaDevices(sess, username)
	if err != nil {
		return "", err
	}

	if len(devices) != 1 {
		return "", fmt.Errorf("unexpected number of mfa devices found, expected 1 was %d", len(devices))
	}

	return *devices[0].SerialNumber, nil
}

func requestNewSTSToken(sess *session.Session, username, mfaToken string, expiry time.Duration, profile string) (*Credentials, error) {
	serial, err := mfaSerialNumber(sess, username)
	if err != nil {
		return nil, err
	}

	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(int64(expiry.Seconds())),
		SerialNumber:    aws.String(serial),
		TokenCode:       aws.String(mfaToken),
	}
	session, err := newLimitedAccessSession(profile)
	if err != nil {
		return nil, err
	}
	svc := sts.New(session, &aws.Config{Region: aws.String("eu-west-1")})
	out, err := svc.GetSessionToken(input)
	if err != nil {
		return nil, err
	}

	return &Credentials{
		AccessKey:    *out.Credentials.AccessKeyId,
		SecretKey:    *out.Credentials.SecretAccessKey,
		SessionToken: *out.Credentials.SessionToken,
		Expiry:       *out.Credentials.Expiration,
	}, nil
}

type Credentials struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
	Expiry       time.Time
}

func (c *Credentials) String() string {
	return fmt.Sprintf("Access Key: %s\nSecret Key: %s\nSession Token: %s\n", c.AccessKey, c.SecretKey, c.SessionToken)
}

func envVarExportsOutput(c *Credentials) {
	fmt.Println()
	fmt.Printf("export AWS_ACCESS_KEY_ID=\"%s\"\n", c.AccessKey)
	fmt.Printf("export AWS_SECRET_ACCESS_KEY=\"%s\"\n", c.SecretKey)
	fmt.Printf("export AWS_SESSION_TOKEN=\"%s\"\n", c.SessionToken)
}

type CredsWriter struct {
	path string
}

func (w *CredsWriter) Output(c *Credentials, profile string) error {
	cfg, err := ini.Load(w.path)
	if err != nil {
		return err
	}

	sec, err := cfg.NewSection(profile)
	if err != nil {
		return err
	}

	_, err = sec.NewKey("aws_access_key_id", c.AccessKey)
	if err != nil {
		return err
	}
	_, err = sec.NewKey("aws_secret_access_key", c.SecretKey)
	if err != nil {
		return err
	}
	_, err = sec.NewKey("aws_session_token", c.SessionToken)
	if err != nil {
		return err
	}

	return cfg.SaveTo(w.path)
}

func newCredentialsFileWriter(credentialsPath string) *CredsWriter {
	return &CredsWriter{credentialsPath}
}

type Expiration struct {
	Expiry time.Time
}

func readExpiry() (*Expiration, error) {
	path, err := expiryPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	dec := gob.NewDecoder(r)

	var ex Expiration
	err = dec.Decode(&ex)
	if err != nil {
		return nil, err
	}

	return &ex, nil
}

func writeExpiry(creds *Credentials) error {
	path, err := expiryPath()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	enc := gob.NewEncoder(w)
	err = enc.Encode(&Expiration{creds.Expiry})
	if err != nil {
		return err
	}
	return w.Flush()
}
