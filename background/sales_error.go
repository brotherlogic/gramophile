package background

import (
	"context"
	"fmt"
	"log"

	ghbclient "github.com/brotherlogic/githubridge/client"
	ghbpb "github.com/brotherlogic/githubridge/proto"
)

func (b *BackgroundRunner) getGHClient() (ghbclient.GithubridgeClient, error) {
	if b.ghclient != nil {
		return b.ghclient, nil
	}
	return ghbclient.GetClientInternal()
}

func (b *BackgroundRunner) reportSaleAdjustmentError(ctx context.Context, sid int64, action string, err error) error {
	client, gerr := b.getGHClient()
	if gerr != nil {
		log.Printf("unable to get githubridge client to report sale adjustment error for sale %v: %v", sid, gerr)
		return nil
	}

	title := fmt.Sprintf("Sale Adjustment Failure: %v on sale %v", action, sid)

	// Deduplication check: check if an open issue with the same title already exists
	resp, gerr := client.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if gerr != nil {
		log.Printf("unable to list issues for deduplication check on sale %v: %v", sid, gerr)
	} else {
		for _, issue := range resp.GetIssues() {
			if issue.GetTitle() == title && issue.GetState() != ghbpb.IssueState_ISSUE_STATE_CLOSED {
				log.Printf("open issue already exists for sale adjustment failure on sale %v: %v (id: %v), skipping creation", sid, title, issue.GetId())
				return nil
			}
		}
	}

	body := fmt.Sprintf("Sale adjustment encountered an unexpected error for sale %v during action %q:\n\n```\n%v\n```", sid, action, err)

	_, gerr = client.CreateIssue(ctx, &ghbpb.CreateIssueRequest{
		User:  "brotherlogic",
		Repo:  "gramophile",
		Title: title,
		Body:  body,
	})
	if gerr != nil {
		log.Printf("unable to create github issue for sale adjustment failure on sale %v: %v", sid, gerr)
		return nil
	}

	return nil
}
