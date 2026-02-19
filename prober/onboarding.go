package main

import (
	"context"
	"fmt"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
)

const (
	userid   = 12345
	username = "simonallantucker"
)

type onboarding struct{}

func (o *onboarding) getName() string {
	return "onboarding"
}

func (o *onboarding) getFrequency() time.Duration {
	return time.Minute * 10
}

func (o *onboarding) runProbe(ctx context.Context, client pb.GramophileEServiceClient, iclient pb.GramophileServiceClient) error {
	userid, err := GetContextKey(ctx)
	if err != nil {
		return err
	}

	// Delete user data
	_, err = iclient.DeleteUser(ctx, &pb.DeleteUserRequest{Id: userid, SoftDelete: true})
	if err != nil {
		return err
	}

	// Reset user
	token := fmt.Sprintf("probertoken-%v", time.Now().UnixNano())
	_, err = iclient.UpgradeUser(ctx, &pb.UpgradeUserRequest{Username: username, NewState: pb.StoredUser_USER_STATE_UNKNOWN, Token: token})
	if err != nil {
		return err
	}

	// Trigger user addition
	_, err = client.GetLogin(ctx, &pb.GetLoginRequest{Token: token})
	if err != nil {
		return err
	}

	// Wait for user to be placed in the waiting list
	user, err := client.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		return err
	}

	if user.GetUser().GetState() != pb.StoredUser_USER_STATE_REFRESHING && user.GetUser().GetState() != pb.StoredUser_USER_STATE_IN_WAITLIST {
		return fmt.Errorf("User is in the wrong state: %v", user.GetUser())
	}

	// Give it five minutes
	t := time.Now()
	for time.Since(t) < time.Minute*5 {
		time.Sleep(time.Second * 10)

		user, err = client.GetUser(ctx, &pb.GetUserRequest{})
		if err != nil {
			return err
		}

		if user.GetUser().GetState() == pb.StoredUser_USER_STATE_IN_WAITLIST {
			break
		}
	}

	user, err = client.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		return err
	}
	if user.GetUser().GetState() != pb.StoredUser_USER_STATE_IN_WAITLIST {
		return fmt.Errorf("User is not in the waitlist: %v", err)
	}

	records, err := client.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "all"})
	if err != nil {
		return err
	}

	if len(records.GetSnapshot().GetPlacements()) < 26 {
		return fmt.Errorf("User collection is too small: %v", len(records.GetSnapshot().GetPlacements()))
	}

	return nil
}
