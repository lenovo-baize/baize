package api

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lenovo-baize/baize/context"
	"github.com/lenovo-baize/baize/data/gather"
	"github.com/lenovo-baize/baize/ipfs/ipfsMain"
)

func reportStop(err error) {
	if !gather.IsReportStarted {
		return
	}
	dataMap := make(map[string]string)
	dataMap[gather.EventActionKey] = "ipfs_daemon_stop"
	if nil != err {
		dataMap["err"] = "daemonErr:" + baizectx.GetContext().DaemonErr + " err:" + err.Error()
	}
	dataMap["err_file"] = getErrorFileContent()
	gather.Gather(dataMap)
	//想要尽量保证最后的数据能上报，所以等待20s
	fmt.Println("wait 10 second to recycle resource before stop")
	time.Sleep(10 * time.Second)
	defer baizectx.GetContext().BaizeCtxCacelFun()
}

//Execute 执行入口
//cmdline命令
//repoPath仓库路径
//params 其他参数
func Execute(cmdline string, repoPath string, params string) (errrs error) {
	baizectx.StartingLock.Lock()
	defer func() {
		if !baizectx.StartingLockIsUnlocked {
			baizectx.StartingLock.Unlock()
		}
		if r := recover(); r != nil {
			errrs = errors.New("Unknow panic")
			switch x := r.(type) {
			case string:
				errrs = errors.New(x)
			case error:
				errrs = x
			default:
			}
			fmt.Fprintf(os.Stderr, "error: %s\n", errrs)
		}
	}()
	if "" == cmdline {
		return errors.New("param cmdline cannot empty")
	}
	if "" == repoPath {
		return errors.New("param repoPath cannot empty")
	}
	os.Setenv("IPFS_PATH", repoPath)
	os.Args = strings.Split(cmdline, " ")
	if isRunDaemon() {
		daemonParams := make(map[string]string)
		daemonParams["repoPath"] = repoPath
		daemonParams["cmdline"] = cmdline
		daemonParams["logPath"] = repoPath + "/log"
		daemonParams["startParams"] = params
		err := runDaemon(daemonParams)
		reportStop(err)
		return err
	}
	return ipfsmain.IpfsMain(context.Background())

}
func getErrorFileContent() string {
	filePath := filepath.Join(baizectx.GetContext().LogPath, "baizeError.log")
	_, err := os.Lstat(filePath)
	if nil != err {
		return "read file error:" + err.Error()
	}
	f, err := os.Open(filePath)
	if err != nil {
		return "read file error:" + err.Error()
	}
	defer f.Close()
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return "read file error:" + err.Error()
	}
	return string(content)
}

//GetNodeID 获取本机节点id,需要节点在启动中，或启动完成后 上下文对象中有了RepoPath后，才能获取
func GetNodeID() (string, error) {
	baizectx.StartingLock.Lock()
	defer baizectx.StartingLock.Unlock()
	peerID := baizectx.GetPeerID()
	if "" == peerID {
		return "", errors.New("baize daemon may be not start or start error")
	}
	return peerID, nil

}

//SetDynamicParam 设置一些动态参数，随着运行会变化的数据，比如网络切换
func SetDynamicParam(key string, value string) {
	baizectx.StartingLock.Lock()
	defer baizectx.StartingLock.Unlock()
	ctx := baizectx.GetContext()
	if nil == ctx {
		return
	}

	if nil == ctx.DynamicParams {
		return
	}
	ctx.SetDynamicParamLock.Lock()
	defer ctx.SetDynamicParamLock.Unlock()
	ctx.DynamicParams[key] = value
	baizectx.TriggerEvent(baizectx.DYNAMIC_PARAM_CHANGE)
}

func isRunDaemon() bool {
	for _, arg := range os.Args {
		if arg == "daemon" {
			return true
		}
	}
	return false
}
