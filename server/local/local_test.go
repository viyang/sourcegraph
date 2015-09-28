package local

import (
	"golang.org/x/net/context"
	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph/mock"
	gitmock "sourcegraph.com/sourcegraph/sourcegraph/gitserver/gitpb/mock"
	"sourcegraph.com/sourcegraph/sourcegraph/store"
	"sourcegraph.com/sourcegraph/sourcegraph/store/mockstore"
	"sourcegraph.com/sourcegraph/sourcegraph/svc"
)

// testContext creates a new context.Context for use by tests that has
// all mockstores instantiated.
func testContext() (context.Context, *mocks) {
	var m mocks
	ctx := NewContext(context.Background(), Config{})
	ctx = store.WithStores(ctx, m.stores.Stores())
	ctx = svc.WithServices(ctx, m.servers.servers())
	return ctx, &m
}

type mocks struct {
	stores  mockstore.Stores
	servers mockServers
}

type mockServers struct {
	// TODO(sqs): move this to go-sourcegraph
	Accounts     mock.AccountsServer
	Auth         mock.AuthServer
	Builds       mock.BuildsServer
	Changesets   mock.ChangesetsServer
	Defs         mock.DefsServer
	Deltas       mock.DeltasServer
	GitTransport gitmock.GitTransportServer
	Markdown     mock.MarkdownServer
	MirrorRepos  mock.MirrorReposServer
	Orgs         mock.OrgsServer
	People       mock.PeopleServer
	RepoBadges   mock.RepoBadgesServer
	RepoStatuses mock.RepoStatusesServer
	RepoTree     mock.RepoTreeServer
	Repos        mock.ReposServer
	Search       mock.SearchServer
	Units        mock.UnitsServer
	Users        mock.UsersServer
}

func (s *mockServers) servers() svc.Services {
	return svc.Services{
		Accounts:     &s.Accounts,
		Auth:         &s.Auth,
		Builds:       &s.Builds,
		Defs:         &s.Defs,
		Deltas:       &s.Deltas,
		GitTransport: &s.GitTransport,
		Markdown:     &s.Markdown,
		MirrorRepos:  &s.MirrorRepos,
		Orgs:         &s.Orgs,
		People:       &s.People,
		RepoBadges:   &s.RepoBadges,
		RepoStatuses: &s.RepoStatuses,
		RepoTree:     &s.RepoTree,
		Repos:        &s.Repos,
		Search:       &s.Search,
		Units:        &s.Units,
		Users:        &s.Users,
	}
}
