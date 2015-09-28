package sgx

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/auth/idkey"
	"sourcegraph.com/sourcegraph/sourcegraph/sgx/cli"
	"sourcegraph.com/sourcegraph/sourcegraph/util/timeutil"
	"sourcegraph.com/sqs/pbtypes"
)

func init() {
	g, err := cli.CLI.AddCommand("registered-clients",
		"manage registered API clients",
		"The registered-clients subcommands manage registered API clients.",
		&regClientsCmd{},
	)
	if err != nil {
		log.Fatal(err)
	}
	g.Aliases = []string{"clients", "rc"}

	_, err = g.AddCommand("create",
		"create (register) an API client",
		"The create subcommand creates (registers) an API clients.",
		&regClientsCreateCmd{},
	)
	if err != nil {
		log.Fatal(err)
	}

	c, err := g.AddCommand("list",
		"list registered API clients",
		"The list subcommand lists registered API clients.",
		&regClientsListCmd{},
	)
	if err != nil {
		log.Fatal(err)
	}
	c.Aliases = []string{"ls"}

	_, err = g.AddCommand("current",
		"gets info about the currently authenticated registered API client",
		"The current subcommand gets info about the currently authenticated registered API client.",
		&regClientsCurrentCmd{},
	)
	if err != nil {
		log.Fatal(err)
	}

	c, err = g.AddCommand("update",
		"updates a registered API client",
		"The update subcommand updates a registered API client.",
		&regClientsUpdateCmd{},
	)
	if err != nil {
		log.Fatal(err)
	}

	c, err = g.AddCommand("delete",
		"deletes a registered API client",
		"The rm subcommand deletes a registered API client.",
		&regClientsDeleteCmd{},
	)
	if err != nil {
		log.Fatal(err)
	}
	c.Aliases = []string{"rm"}
}

type regClientsCmd struct{}

func (*regClientsCmd) Execute(args []string) error { return nil }

type regClientsListCmd struct {
	Detail bool `short:"d" long:"detail" description:"show full details"`
}

func (c *regClientsListCmd) Execute(args []string) error {
	cl := Client()

	opt := sourcegraph.RegisteredClientListOptions{
		ListOptions: sourcegraph.ListOptions{Page: 1},
	}
	for {
		clients, err := cl.RegisteredClients.List(cliCtx, &opt)
		if err != nil {
			return err
		}
		for _, client := range clients.Clients {
			if c.Detail {
				printRegisteredClient(client)
			} else {
				users, err := cl.RegisteredClients.ListUserPermissions(cliCtx, &sourcegraph.RegisteredClientSpec{ID: client.ID})
				if grpc.Code(err) == codes.PermissionDenied {
					err = nil
					users = &sourcegraph.UserPermissionsList{}
				} else if err != nil {
					return err
				}

				var host string
				if len(client.RedirectURIs) > 0 {
					url, err := url.Parse(client.RedirectURIs[0])
					if err != nil {
						host = "(invalid URL)"
					} else {
						host = url.Host
					}
				} else {
					host = "(none)"
				}

				fmt.Printf("%- 25s   %d users   %s   %- 20s\n", host, len(users.UserPermissions), timeutil.TimeAgo(client.CreatedAt), client.ClientName)
				for _, up := range users.UserPermissions {
					u, err := cl.Users.Get(cliCtx, &sourcegraph.UserSpec{UID: up.UID})
					if err != nil {
						return err
					}
					fmt.Printf("\t%s (%s)\n", u.Login, accessString(up))
				}
			}
		}
		if !clients.HasMore {
			break
		}
		opt.Page++
	}
	return nil
}

type regClientsCreateCmd struct {
	ClientName  string `long:"client-name"`
	ClientURI   string `long:"client-uri"`
	RedirectURI string `long:"redirect-uri"`
	Description string `long:"description"`
	Type        string `long:"type" default:"SourcegraphServer"`
	IDKeyFile   string `short:"i" long:"id-key-file" description:"path to file containing ID key (only public key is transmitted)" default:"$SGPATH/id.pem"`
}

func (c *regClientsCreateCmd) Execute(args []string) error {
	cl := Client()

	typ, ok := sourcegraph.RegisteredClientType_value[c.Type]
	if !ok {
		return fmt.Errorf("invalid --type %q; choices are %+v", c.Type, sourcegraph.RegisteredClientType_value)
	}

	c.IDKeyFile = os.ExpandEnv(c.IDKeyFile)
	data, err := ioutil.ReadFile(c.IDKeyFile)
	if err != nil {
		return err
	}
	idKey, err := idkey.New(data)
	if err != nil {
		return err
	}
	log.Printf("# Using public key from file %s", c.IDKeyFile)
	jwks, err := idKey.MarshalJWKSPublicKey()
	if err != nil {
		return err
	}

	regClient := &sourcegraph.RegisteredClient{
		ID:          idKey.ID,
		ClientName:  c.ClientName,
		ClientURI:   c.ClientURI,
		JWKS:        string(jwks),
		Description: c.Description,
		Type:        sourcegraph.RegisteredClientType(typ),
	}
	if c.RedirectURI != "" {
		regClient.RedirectURIs = []string{c.RedirectURI}
	}

	regClient, err = cl.RegisteredClients.Create(cliCtx, regClient)
	if err != nil {
		return err
	}

	log.Println("# Registered API client:")
	printRegisteredClient(regClient)
	return nil
}

type regClientsCurrentCmd struct{}

func (c *regClientsCurrentCmd) Execute(args []string) error {
	cl := Client()

	client, err := cl.RegisteredClients.GetCurrent(cliCtx, &pbtypes.Void{})
	if err != nil {
		return err
	}
	printRegisteredClient(client)
	return nil
}

func printRegisteredClient(c *sourcegraph.RegisteredClient) {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

type regClientsUpdateCmd struct {
	ClientName  string `long:"client-name"`
	ClientURI   string `long:"client-uri"`
	RedirectURI string `long:"redirect-uri"`
	Description string `long:"description"`
	AllowLogins string `long:"allow-logins" description:"set to 'all' to allow any user to login to this client" default:"restricted"`

	Args struct {
		ClientID string `name:"CLIENT-ID"`
	} `positional-args:"yes" count:"1"`
}

func (c *regClientsUpdateCmd) Execute(args []string) error {
	cl := Client()

	rc, err := cl.RegisteredClients.Get(cliCtx, &sourcegraph.RegisteredClientSpec{ID: c.Args.ClientID})
	if err != nil {
		return err
	}

	fmt.Print(rc.ID, ": ")
	if c.ClientName != "" {
		rc.ClientName = c.ClientName
	}
	if c.ClientURI != "" {
		rc.ClientURI = c.ClientURI
	}
	if c.RedirectURI != "" {
		rc.RedirectURIs = []string{c.RedirectURI}
	}
	if c.Description != "" {
		rc.Description = c.Description
	}
	if c.AllowLogins != "" {
		if rc.Meta == nil {
			rc.Meta = map[string]string{}
		}
		rc.Meta["allow-logins"] = c.AllowLogins
	}
	if _, err := cl.RegisteredClients.Update(cliCtx, rc); err != nil {
		return err
	}
	fmt.Println("updated")
	return nil
}

type regClientsDeleteCmd struct {
	Args struct {
		ClientIDs []string `name:"CLIENT-IDs"`
	} `positional-args:"yes"`
}

func (c *regClientsDeleteCmd) Execute(args []string) error {
	cl := Client()

	for _, clientID := range c.Args.ClientIDs {
		fmt.Print(clientID, ": ")
		if _, err := cl.RegisteredClients.Delete(cliCtx, &sourcegraph.RegisteredClientSpec{ID: clientID}); err != nil {
			return err
		}
		fmt.Println("deleted")
	}
	return nil
}
