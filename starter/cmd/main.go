package main

import (
	"context"
	"fmt"
	"baize/starter/api"
	"github.com/ipfs/go-ipfs-cmds"
	"github.com/ipfs/go-ipfs-cmds/cli"
	"os"

	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"baize/ipfs/ipfsMain"
)

func main() {
	req, errParse := cli.Parse(context.Background(), os.Args[1:], os.Stdin, ipfsmain.Root)
	if nil != errParse {
		fmt.Fprintf(os.Stderr, "Error: %s\n", errParse)
		return
	}
	repoPath, err := getRepoPath(req)
	if err != nil {
		os.Exit(1)
	}
	cmdline := ""
	for _, arg := range os.Args {
		if "" == cmdline {
			cmdline = arg
		} else {
			cmdline = cmdline + " " + arg
		}

	}
	e := api.Execute(cmdline, repoPath, "source:cmdline")

	//e := api.Execute(cmdline, repoPath, "source:android-phone|network:WIFI")
	if e != nil {
		os.Exit(1)
	}
	os.Exit(0)

}
func getRepoPath(req *cmds.Request) (string, error) {
	repoOpt, found := req.Options["config"].(string)
	if found && repoOpt != "" {
		return repoOpt, nil
	}

	repoPath, err := fsrepo.BestKnownPath()
	if err != nil {
		return "", err
	}
	return repoPath, nil
}
