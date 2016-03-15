package gitserver

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"net/rpc"
	"os"
	"os/exec"
	"path"

	"src.sourcegraph.com/sourcegraph/pkg/vcs"
)

type CreateArgs struct {
	Repo         string
	MirrorRemote string
	Opt          *vcs.RemoteOpts
}

func (g *Git) Create(args *CreateArgs, reply *struct{}) error {
	dir := path.Join(ReposDir, args.Repo)

	if args.MirrorRemote != "" {
		cmd := exec.Command("git", "clone", "--mirror", args.MirrorRemote, dir)

		var outputBuf bytes.Buffer
		cmd.Stdout = &outputBuf
		cmd.Stderr = &outputBuf
		if err := runWithRemoteOpts(cmd, args.Opt); err != nil {
			return fmt.Errorf("cloning repository %s failed with output:\n%s", args.Repo, outputBuf.String())
		}
		return nil
	}

	cmd := exec.Command("git", "init", "--bare", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("initializing repository %s failed with output:\n%s", args.Repo, string(out))
	}
	return nil
}

func Init(repo string) error {
	return create(repo, "", nil)
}

func Clone(repo string, remote string, opt *vcs.RemoteOpts) error {
	if remote == "" {
		return errors.New("empty remote")
	}
	return create(repo, remote, opt)
}

// create creates a new repository in the gitserver cluster by initializing an empty repository
// if mirrorRemote is empty or clones the given remote otherwise, using opt for authentication.
// The gitserver is selected pseudo-randomly.
func create(repo string, mirrorRemote string, opt *vcs.RemoteOpts) error {
	cmd := Command("git", "remote")
	cmd.Repo = repo
	err := cmd.Run()
	if err == nil {
		return errors.New("repository already exists")
	}
	if err != vcs.ErrRepoNotExist {
		return err
	}

	// This hash is used to avoid concurrent init on two servers, it does not need to be stable over long timespans.
	h := fnv.New32a()
	if _, err := h.Write([]byte(repo)); err != nil {
		return err
	}

	sum := md5.Sum([]byte(repo))
	serverIndex := binary.BigEndian.Uint64(sum[:]) % uint64(len(servers))

	done := make(chan *rpc.Call, 1)
	servers[serverIndex] <- &rpc.Call{
		ServiceMethod: "Git.Create",
		Args:          &CreateArgs{Repo: repo, MirrorRemote: mirrorRemote, Opt: opt},
		Reply:         &struct{}{},
		Done:          done,
	}
	return (<-done).Error
}

type RemoveArgs struct {
	Repo string
}

type RemoveReply struct {
	RepoExists bool
}

func (r *RemoveReply) repoExists() bool {
	return r.RepoExists
}

func (g *Git) Remove(args *RemoveArgs, reply *RemoveReply) error {
	dir := path.Join(ReposDir, args.Repo)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	reply.RepoExists = true

	cmd := exec.Command("git", "remote")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a repository: %s", args.Repo)
	}

	return os.RemoveAll(dir)
}

func Remove(repo string) error {
	_, err := broadcastCall(
		"Git.Remove",
		&RemoveArgs{Repo: repo},
		func() repoExistsReply { return new(RemoveReply) },
	)
	return err
}
