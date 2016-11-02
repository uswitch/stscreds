package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"time"
)

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
	svc := sts.New(sess, &aws.Config{Region: aws.String("eu-west-1")})
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
