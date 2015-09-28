// generated by gen-mocks; DO NOT EDIT

package mockstore

import (
	"time"

	"golang.org/x/net/context"
	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/store"
)

type Authorizations struct {
	CreateAuthCode_ func(ctx context.Context, req *sourcegraph.AuthorizationCodeRequest, expires time.Duration) (string, error)
	MarkExchanged_  func(ctx context.Context, code *sourcegraph.AuthorizationCode, clientID string) (*sourcegraph.AuthorizationCodeRequest, error)
}

func (s *Authorizations) CreateAuthCode(ctx context.Context, req *sourcegraph.AuthorizationCodeRequest, expires time.Duration) (string, error) {
	return s.CreateAuthCode_(ctx, req, expires)
}

func (s *Authorizations) MarkExchanged(ctx context.Context, code *sourcegraph.AuthorizationCode, clientID string) (*sourcegraph.AuthorizationCodeRequest, error) {
	return s.MarkExchanged_(ctx, code, clientID)
}

var _ store.Authorizations = (*Authorizations)(nil)
