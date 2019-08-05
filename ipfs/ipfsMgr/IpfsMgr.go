package ipfsmgr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lenovo-baize/baize/context"
	"github.com/lenovo-baize/baize/ipfs/ipfsMain"
)

//StartIpfs 启动ipfs
func StartIpfs() error {
	//删除api文件，从灰度日志看有因为该文件中记录的地址不正确，导致启动失败的问题
	delAPIFile()
	ipfsCtx, ipfsCtxCacelFun := context.WithCancel(baizectx.GetContext().BaizeCtx)
	baizectx.GetContext().IpfsCtx = ipfsCtx
	baizectx.GetContext().IpfsCtxCacelFun = ipfsCtxCacelFun
	err := ipfsmain.IpfsMain(baizectx.GetContext().IpfsCtx)
	baizectx.GetContext().IsIpfsStart = false
	if nil != err {
		return errors.New("daemonErr:" + baizectx.GetContext().DaemonErr + " err:" + err.Error())
	}
	return nil

}

//delAPIFile 删除daemon时生成的api文件，从灰度日志看有因为该文件中记录的地址不正确，导致启动失败的问题
func delAPIFile() {
	apiPath := filepath.Join(baizectx.GetContext().RepoPath, "api")
	_, err := os.Stat(apiPath)
	if os.IsNotExist(err) {
		return
	}
	err = os.Remove(apiPath)
	if nil != err {
		fmt.Fprintf(os.Stderr, "remove api file error,path: %s,error:%s\n", apiPath, err)
	}
}
