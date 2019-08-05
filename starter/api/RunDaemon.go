package api

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"baize/config/cfgmgr"
	"baize/context"
	"baize/data"
	"baize/ipfs/ipfsMgr"

	runmodedetect "baize/runmode/runmodedetect"
	"os"
	"path/filepath"
	"strings"
)

func runDaemon(params map[string]string) (errrs error) {
	defer func() {
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

	//仓库检查，如果当前节点没有启动成功后，删除仓库重建,从灰度数据看，没有启动成功，有没有初始化完成的可能
	repoCheck(params["repoPath"])

	setLogPath(params["logPath"])

	err := baizectx.InitBaizeContext(params)
	if nil != err {
		fmt.Fprintf(os.Stderr, "init baize context error:%s\n", err)
		return errors.New("init baize context error: " + err.Error())
	}
	//初始化白泽配置，第1次使用默认配置，后续使用本地缓存的配置
	err = cfgmgr.InitConfig()
	if nil != err {
		fmt.Fprintf(os.Stderr, "error init baize config,nodeid:%s,err:%s\n", baizectx.GetPeerID(), err)
		return errors.New("Error init baize config nodeid:" + baizectx.GetPeerID() + ",err: " + err.Error())
	}
	//集群密钥文件如果没有创建，先创建
	err = genSwarmKeyFile()
	if nil != err {
		fmt.Fprintf(os.Stderr, "error create swarm.key nodeid:%s,error:%s\n", baizectx.GetPeerID(), err)
		return errors.New("Error create swarm.key nodeid:" + baizectx.GetPeerID() + ",err: " + err.Error())
	}

	//初始化数据采集
	err = data.Init()
	if nil != err {
		fmt.Fprintf(os.Stderr, "Error init DataReport nodeid:%s,err: %s\n", baizectx.GetPeerID(), err)
		return errors.New("Error init DataReport nodeid:" + baizectx.GetPeerID() + ",err: " + err.Error())
	}
	cmdline := baizectx.GetContext().Cmdline
	//ipfs的启动命令中增加:
	//init选项，当仓库没有初始化时先初始化，初始化后会根据白泽的配置同步一次ipfs的配置
	//enable-gc 启动block回收功能
	cmdline = cmdline + " --init --enable-gc"
	os.Args = strings.Split(cmdline, " ")
	//监听启动完成事件，启动后从服务端同步1次配置
	baizectx.ListenEvent(baizectx.EVENT_START, func() {
		e := cfgmgr.SyncServerConfig()
		if e != nil {
			fmt.Fprintf(os.Stderr, "Error SyncServerConfig nodeid:%s,err: %s\n", baizectx.GetPeerID(), e)
		}
	})
	//启动定时从服务端同步白泽配置,同步成功会将新的配置缓存到本地
	go cfgmgr.Time2SyncConfig()
	//选择运行模式
	runmodedetect.DetectRunModeBeforeStart()
	//启动ipfs节点，启动成功，将hold住当前线程，直到退出
	err = ipfsmgr.StartIpfs()
	if nil != err {
		fmt.Fprintf(os.Stderr, "Error start baize nodeid:%s  %s\n", baizectx.GetPeerID(), err)
		return errors.New("Error start baize nodeid:" + baizectx.GetPeerID() + " " + err.Error())
	}
	return nil
}
func repoCheck(reporPath string) {
	if !fsrepo.IsInitialized(reporPath) {
		err := os.RemoveAll(reporPath)
		if nil != err {
			fmt.Fprintf(os.Stderr, "erro	remove repo to  init 1,err:%s\n", err)
		}
		return
	}
	//启动成功标志文件
	startedFlagPath := filepath.Join(filepath.Join(reporPath, "baize"), "start-success")
	_, err := os.Stat(startedFlagPath)
	//没有启动成功，仓库删除重建,没有成功启动，不确定上传初始化的仓库是否是正确的，从灰度看，有没有初始化完成的可能
	if nil != err {
		err = os.RemoveAll(reporPath)
		if nil != err {
			fmt.Fprintf(os.Stderr, "erro	 remove repo to  init 2,err:%s\n", err)
		}
	}
}

func setLogPath(logPath string) {
	if "" == logPath {
		return
	}
	_, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(logPath, 0755)
	}
	stdOutFile, e := os.OpenFile(filepath.Join(logPath, "baizeStd.log"), os.O_RDWR|os.O_CREATE|os.O_SYNC|os.O_TRUNC, 0600)
	if nil != e {
		fmt.Fprintf(os.Stderr, "create baizeStd error,path: %s,error:%s\n", logPath, err)
	} else {
		os.Stdout = stdOutFile
	}
	errorOutFile, e := os.OpenFile(filepath.Join(logPath, "baizeError.log"), os.O_RDWR|os.O_CREATE|os.O_SYNC|os.O_TRUNC, 0600)
	if nil != e {
		fmt.Fprintf(os.Stderr, "create baizeError error,path: %s,error:%s\n", logPath, err)
	} else {
		os.Stderr = errorOutFile
	}
	// 将进程标准出错重定向至文件，进程崩溃时运行时将向该文件记录协程调用栈信息
	//syscall.Dup2(int(errorOutFile.Fd()), int(os.Stderr.Fd()))
	//fmt.Println(os.Stdout, "syscall.Dup2 changer to baizeError.log")
}
func genSwarmKeyFile() error {
	swarmKeyPath := filepath.Join(baizectx.GetContext().RepoPath, "swarm.key")
	_, err := os.Stat(swarmKeyPath)
	if os.IsNotExist(err) {
		swarmKeyFile, err := os.Create(swarmKeyPath)
		if err != nil {
			panic("error create swarm.key file:" + err.Error())
		}
		defer swarmKeyFile.Close()
		w := bufio.NewWriter(swarmKeyFile)

		fmt.Fprintln(w, "/key/swarm/psk/1.0.0/")
		fmt.Fprintln(w, "/base16/")
		//当前线上key
		fmt.Fprintln(w, "6d62b1b2fdbbbe0544a036615263e74c22e177902fecefdcea67c5bf526f4e39")
		//测试环境key
		//fmt.Fprintln(w, "386ea607a38453824fc10e0df7062257226b31ff4eb5151c018a3284ec28c390")
		err = w.Flush()
		if err != nil {
			//写失败了，将文件删除
			os.Remove(swarmKeyFile.Name())
			return errors.New("error create swarm.key path:" + swarmKeyPath + " err:" + err.Error())
		}
	}
	return nil
}
