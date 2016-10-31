package main

import (
	"fmt"
)

type WhoAmI struct {
	Profile string
}

func (w *WhoAmI) Execute() error {
	sess, err := newLimitedAccessSession(w.Profile)
	if err != nil {
		return err
	}

	user, err := getUser(sess)
	if err != nil {
		return err
	}

	devices, err := mfaDevices(sess, *user.UserName)
	if err != nil {
		return err
	}

	fmt.Printf("%+v\n", user)
	for _, device := range devices {
		fmt.Printf("%+v\n", device)
	}
	return nil
}
