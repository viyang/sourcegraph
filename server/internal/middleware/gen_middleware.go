// +build ignore

package main

import (
	"go/ast"
	"text/template"

	"sourcegraph.com/sourcegraph/sourcegraph/gen"
)

func main() {
	svcs := []string{
		"../../../Godeps/_workspace/src/sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph/sourcegraph.pb.go",
		"../../../Godeps/_workspace/src/sourcegraph.com/sourcegraph/srclib/store/pb/srcstore.pb.go",
		"../../../gitserver/gitpb/git_transport.pb.go",
	}
	gen.Generate("middleware.go", tmpl, svcs, nil)
}

func serviceIsFederated(x *gen.Service) bool {
	switch x.Name {
	case "Accounts", "Auth", "Builds", "Defs", "Deltas", "MirrorRepos", "RegisteredClients", "RepoBadges", "RepoStatuses", "RepoTree", "Repos", "Search", "Units", "Users":
		return true
	}
	return false
}

func methodHasCustomFederation(x gen.Service, method string) bool {
	switch x.Name {
	case "Accounts":
		return method == "Update"
	case "Auth":
		switch method {
		case "GetAccessToken", "Identify":
			return true
		default:
			return false
		}
	case "Builds":
		return method == "List"
	case "Defs":
		return method == "List"
	case "Repos":
		switch method {
		case "Create", "Get", "List":
			return true
		default:
			return false
		}
	case "MirrorRepos":
		return method == "RefreshVCS"
	case "Users":
		switch method {
		case "Get", "CheckWhitelist", "AddToWhitelist":
			return true
		default:
			return false
		}
	case "RegisteredClients":
		// hack: keep every method with custom implementation
		return true
	default:
		return false
	}
}

func repoURIExpr(m *gen.Method) string {
	expr := gen.RepoURIExpr(ast.NewIdent("param"), m.Type.Params.List[1].Type)
	if expr == nil {
		return ""
	}
	return gen.AstString(expr)
}

func userSpecExpr(m *gen.Method) string {
	expr := gen.UserSpecExpr(ast.NewIdent("param"), m.Type.Params.List[1].Type)
	if expr == nil {
		return ""
	}
	return gen.AstString(expr)
}

var tmpl = template.Must(template.New("").Delims("<<<", ">>>").Funcs(template.FuncMap{
	"serviceIsFederated":        serviceIsFederated,
	"methodHasCustomFederation": methodHasCustomFederation,
	"repoURIExpr":               repoURIExpr,
	"userSpecExpr":              userSpecExpr,
}).Parse(`// GENERATED CODE - DO NOT EDIT!
//
// Generated by:
//
//   go run gen_middleware.go
//
// Called via:
//
//   go generate
//

package middleware

import (
	"time"

	"golang.org/x/net/context"
	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/sourcegraph/gitserver/gitpb"
	"sourcegraph.com/sourcegraph/sourcegraph/server/internal/middleware/auth"
	"sourcegraph.com/sourcegraph/sourcegraph/server/internal/middleware/federated"
	"sourcegraph.com/sourcegraph/sourcegraph/server/internal/middleware/trace"
	"sourcegraph.com/sourcegraph/sourcegraph/svc"
	"sourcegraph.com/sourcegraph/srclib/store/pb"
	"sourcegraph.com/sourcegraph/srclib/unit"
	"sourcegraph.com/sqs/pbtypes"
)

// Wrap wraps the services and returns a set of services that performs
// authorization checks as specified in the Config.
func Wrap(s svc.Services, c *auth.Config) svc.Services {
	<<<range .>>>
	  if s.<<<.Name>>> != nil {
			s.<<<.Name>>> = wrapped<<<.Name>>>{s.<<<.Name>>>, c}
		}
	<<<end>>>
	return s
}

<<<range .>>>
	type wrapped<<<.Name>>> struct{
		u <<<.TypeName>>>
		c *auth.Config
	}

  <<<$service := .>>>
	<<<range .Methods>>>
		func (s wrapped<<<$service.Name>>>) <<<.Name>>>(ctx context.Context, param <<<.ParamType>>>) (res <<<.ResultType>>>, err error) {
			start := time.Now()
			ctx = trace.Before(ctx, "<<<$service.Name>>>", "<<<.Name>>>", param)
			defer func(){
		  	trace.After(ctx, "<<<$service.Name>>>", "<<<.Name>>>", param, err, time.Since(start))
			}()

			err = s.c.Authenticate(ctx, "<<<$service.Name>>>.<<<.Name>>>")
			if err != nil {
				return
			}

			<<<if methodHasCustomFederation $service .Name>>>
				res, err = federated.Custom<<<$service.Name>>><<<.Name>>>(ctx, param, s.u)
				return
			<<<else>>>
				var target <<<$service.TypeName>>> = s.u
				<<<$repoURIExpr := repoURIExpr .>>>
				<<<$userSpecExpr := userSpecExpr .>>>
				<<<if and (serviceIsFederated $service) (or (ne $repoURIExpr "") (ne $userSpecExpr ""))>>>
					var fedCtx context.Context
					fedCtx, err = <<<if ne $repoURIExpr "">>>federated.RepoContext(ctx, &<<<$repoURIExpr>>>)<<<else if ne $userSpecExpr "">>>federated.UserContext(ctx, <<<$userSpecExpr>>>)<<<end>>>
					if err != nil {
						return
					}
					if fedCtx != nil {
						target = svc.<<<$service.Name>>>(fedCtx)
						ctx = fedCtx
					}
				<<<end>>>

				res, err = target.<<<.Name>>>(ctx, param)
				return
			<<<end>>>
		}
	<<<end>>>
<<<end>>>
`))
