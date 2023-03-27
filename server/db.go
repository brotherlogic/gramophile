package server

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	pb "github.com/brotherlogic/gramophile/proto"
	rspb "github.com/brotherlogic/rstore/proto"
)

type db struct{}

func (d *db) loadLogins(ctx context.Context) (*pb.UserLoginAttempts, error) {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := rspb.NewRStoreServiceClient(conn)
	val, err := client.Read(ctx, &rspb.ReadRequest{
		Key: "gramophile/logins",
	})
	if err != nil {
		// OutOfRange indicates that the key was not found
		if status.Code(err) == codes.OutOfRange {
			return &pb.UserLoginAttempts{}, nil
		}
		return nil, err
	}

	var logins *pb.UserLoginAttempts
	err = proto.Unmarshal(val.GetValue().GetValue(), logins)
	if err != nil {
		return nil, err
	}

	return logins, nil
}

func (d *db) saveLogins(ctx context.Context, logins *pb.UserLoginAttempts) error {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return err
	}

	data, err := proto.Marshal(logins)
	if err != nil {
		return err
	}

	client := rspb.NewRStoreServiceClient(conn)
	_, err = client.Write(ctx, &rspb.WriteRequest{
		Key:   "gramophile/logins",
		Value: &anypb.Any{Value: data},
	})

	return err
}

func (d *db) generateToken(ctx context.Context, token, secret string) (*pb.GramophileAuth, error) {
	user := fmt.Sprintf("%v-%v", time.Now().UnixNano(), rand.Int63())
	su := &pb.StoredUser{
		Auth:       &pb.GramophileAuth{Token: user},
		UserToken:  token,
		UserSecret: secret,
	}

	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	data, err := proto.Marshal(su)
	if err != nil {
		return nil, err
	}

	client := rspb.NewRStoreServiceClient(conn)
	_, err = client.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("gramophile/user/%v", user),
		Value: &anypb.Any{Value: data},
	})

	return &pb.GramophileAuth{Token: user}, nil
}
