package server

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGenerateContext(t *testing.T) {
	ctx := generateContext(context.Background(), "blah")

	key, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("bad extract")
	}
	if key.Get("trace-id")[0] == "" {
		t.Errorf("Bad pull: %v", key)
	}
}

func TestAuthCheck(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	s := BuildServer(d, nil, nil)

	users := []*pb.StoredUser{
		{
			User:  &pbd.User{DiscogsUserId: 1, Username: "live"},
			State: pb.StoredUser_USER_STATE_LIVE,
			Auth:  &pb.GramophileAuth{Token: "token-live"},
		},
		{
			User:  &pbd.User{DiscogsUserId: 2, Username: "waitlist"},
			State: pb.StoredUser_USER_STATE_IN_WAITLIST,
			Auth:  &pb.GramophileAuth{Token: "token-waitlist"},
		},
		{
			User:  &pbd.User{DiscogsUserId: 3, Username: "refreshing"},
			State: pb.StoredUser_USER_STATE_REFRESHING,
			Auth:  &pb.GramophileAuth{Token: "token-refreshing"},
		},
		{
			User:  &pbd.User{DiscogsUserId: 4, Username: "unknown"},
			State: pb.StoredUser_USER_STATE_UNKNOWN,
			Auth:  &pb.GramophileAuth{Token: "token-unknown"},
		},
	}

	for _, u := range users {
		if err := d.SaveUser(context.Background(), u); err != nil {
			t.Fatalf("SaveUser(%v) failed: %v", u.GetAuth().GetToken(), err)
		}
	}

	authCtx := func(token string) context.Context {
		return metadata.NewIncomingContext(context.Background(), metadata.Pairs("auth-token", token))
	}

	cases := []struct {
		name      string
		ctx       context.Context
		method    string
		wantAllow bool
	}{
		// Unauthenticated
		{name: "unauth GetURL", ctx: context.Background(), method: pb.GramophileEService_GetURL_FullMethodName, wantAllow: true},
		{name: "unauth GetLogin", ctx: context.Background(), method: pb.GramophileEService_GetLogin_FullMethodName, wantAllow: true},
		{name: "unauth GetUser", ctx: context.Background(), method: pb.GramophileEService_GetUser_FullMethodName, wantAllow: false},
		{name: "unauth GetState", ctx: context.Background(), method: pb.GramophileEService_GetState_FullMethodName, wantAllow: false},
		{name: "unauth GetOrg", ctx: context.Background(), method: pb.GramophileEService_GetOrg_FullMethodName, wantAllow: false},
		{name: "unauth SetConfig", ctx: context.Background(), method: pb.GramophileEService_SetConfig_FullMethodName, wantAllow: false},

		// Invalid token
		{name: "invalid token GetUser", ctx: authCtx("token-invalid"), method: pb.GramophileEService_GetUser_FullMethodName, wantAllow: false},
		{name: "invalid token GetState", ctx: authCtx("token-invalid"), method: pb.GramophileEService_GetState_FullMethodName, wantAllow: false},
		{name: "invalid token GetOrg", ctx: authCtx("token-invalid"), method: pb.GramophileEService_GetOrg_FullMethodName, wantAllow: false},

		// Waitlisted
		{name: "waitlist GetUser", ctx: authCtx("token-waitlist"), method: pb.GramophileEService_GetUser_FullMethodName, wantAllow: true},
		{name: "waitlist GetState", ctx: authCtx("token-waitlist"), method: pb.GramophileEService_GetState_FullMethodName, wantAllow: true},
		{name: "waitlist GetOrg", ctx: authCtx("token-waitlist"), method: pb.GramophileEService_GetOrg_FullMethodName, wantAllow: false},
		{name: "waitlist SetConfig", ctx: authCtx("token-waitlist"), method: pb.GramophileEService_SetConfig_FullMethodName, wantAllow: false},

		// Refreshing
		{name: "refreshing GetUser", ctx: authCtx("token-refreshing"), method: pb.GramophileEService_GetUser_FullMethodName, wantAllow: true},
		{name: "refreshing GetState", ctx: authCtx("token-refreshing"), method: pb.GramophileEService_GetState_FullMethodName, wantAllow: true},
		{name: "refreshing GetOrg", ctx: authCtx("token-refreshing"), method: pb.GramophileEService_GetOrg_FullMethodName, wantAllow: false},
		{name: "refreshing SetConfig", ctx: authCtx("token-refreshing"), method: pb.GramophileEService_SetConfig_FullMethodName, wantAllow: false},

		// Unknown state
		{name: "unknown GetUser", ctx: authCtx("token-unknown"), method: pb.GramophileEService_GetUser_FullMethodName, wantAllow: true},
		{name: "unknown GetState", ctx: authCtx("token-unknown"), method: pb.GramophileEService_GetState_FullMethodName, wantAllow: true},
		{name: "unknown GetOrg", ctx: authCtx("token-unknown"), method: pb.GramophileEService_GetOrg_FullMethodName, wantAllow: false},
		{name: "unknown SetConfig", ctx: authCtx("token-unknown"), method: pb.GramophileEService_SetConfig_FullMethodName, wantAllow: false},

		// Live user
		{name: "live GetURL", ctx: authCtx("token-live"), method: pb.GramophileEService_GetURL_FullMethodName, wantAllow: true},
		{name: "live GetLogin", ctx: authCtx("token-live"), method: pb.GramophileEService_GetLogin_FullMethodName, wantAllow: true},
		{name: "live GetUser", ctx: authCtx("token-live"), method: pb.GramophileEService_GetUser_FullMethodName, wantAllow: true},
		{name: "live GetState", ctx: authCtx("token-live"), method: pb.GramophileEService_GetState_FullMethodName, wantAllow: true},
		{name: "live GetOrg", ctx: authCtx("token-live"), method: pb.GramophileEService_GetOrg_FullMethodName, wantAllow: true},
		{name: "live SetConfig", ctx: authCtx("token-live"), method: pb.GramophileEService_SetConfig_FullMethodName, wantAllow: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				called = true
				return "ok", nil
			}
			info := &grpc.UnaryServerInfo{FullMethod: tc.method}

			resp, err := s.AuthCheck(tc.ctx, nil, info, handler)
			if tc.wantAllow {
				if err != nil {
					t.Fatalf("AuthCheck(%s) unexpected error: %v", tc.name, err)
				}
				if !called {
					t.Errorf("AuthCheck(%s) did not invoke handler", tc.name)
				}
				if resp != "ok" {
					t.Errorf("AuthCheck(%s) resp = %v, want \"ok\"", tc.name, resp)
				}
				return
			}

			if err == nil {
				t.Fatalf("AuthCheck(%s) expected error, got nil", tc.name)
			}
			if status.Code(err) != codes.Unauthenticated {
				t.Errorf("AuthCheck(%s) code = %v, want %v (err=%v)", tc.name, status.Code(err), codes.Unauthenticated, err)
			}
			if called {
				t.Errorf("AuthCheck(%s) invoked handler when rejected", tc.name)
			}
		})
	}
}
