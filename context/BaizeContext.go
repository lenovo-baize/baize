package baizectx

import (
	"container/list"
	"context"
	"fmt"
	"github.com/lenovo-baize/baize/runmode/runmodestat"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	serialize "github.com/ipfs/go-ipfs-config/serialize"

	"github.com/ipfs/go-ipfs/core"
	gc "github.com/ipfs/go-ipfs/core/corerepo"
	"github.com/lenovo-baize/baize/config"
	"github.com/lenovo-baize/baize/data/gather"
)

var StartingLock sync.Mutex
var StartingLockIsUnlocked bool = false

//BaizeContext 白泽上下文
type BaizeContext struct {
	Cmdline             string
	Version             string
	RepoPath            string
	StartParams         map[string]string
	DynamicParams       map[string]string
	ConfigURLs          []string
	BaizeRootPath       string
	BaizeConfigPath     string
	DataGatherTmpPath   string
	Config              *config.Config
	IpfsPeerID          string
	IpfsNode            *core.IpfsNode
	IsIpfsStart         bool
	IpfsCtx             context.Context
	IpfsCtxCacelFun     context.CancelFunc
	BaizeCtx            context.Context
	BaizeCtxCacelFun    context.CancelFunc
	DaemonErr           string
	Events              map[string]*list.List
	MetricsCount        map[string]int
	LogPath             string
	SetDynamicParamLock sync.Mutex
}

//EVENT_START 启动事件
const EVENT_START = "start"

//DYNAMIC_PARAM_CHANGE 动态参数改变事件
const DYNAMIC_PARAM_CHANGE = "dynamicParamChange"

//白泽模块上下文对象
var contextObj *BaizeContext

//InitBaizeContext 初始化白泽上下文
func InitBaizeContext(params map[string]string) error {

	contextObj = &BaizeContext{Version: "3.0"}
	contextObj.RepoPath = params["repoPath"]
	contextObj.Cmdline = params["cmdline"]
	contextObj.LogPath = params["logPath"]
	// contextObj.ConfigURLs = []string{"http://10.109.18.12:9527/config/get"}
	// contextObj.ConfigURLs = []string{"http://10.109.22.136:8090/config"}
	contextObj.ConfigURLs = []string{"http://ipfsdata.lenovomm.cn/config/get", "http://54.223.108.72:8888/config/get", "http://52.80.252.8:8888/config/get", "http://52.80.179.53:8888/config/get"}

	contextObj.BaizeRootPath = filepath.Join(contextObj.RepoPath, "baize")
	err := creatPathIfNotExist(contextObj.BaizeRootPath)
	if nil != err {
		return err
	}
	contextObj.DataGatherTmpPath = filepath.Join(contextObj.BaizeRootPath, "cache")
	err = creatPathIfNotExist(contextObj.DataGatherTmpPath)
	if nil != err {
		return err
	}

	contextObj.MetricsCount = make(map[string]int)

	contextObj.BaizeConfigPath = filepath.Join(contextObj.BaizeRootPath, "config")
	contextObj.DynamicParams = make(map[string]string)
	contextObj.StartParams = make(map[string]string)
	if "" != params["startParams"] {
		param := strings.Split(params["startParams"], "|")
		for i := 0; i < len(param); i++ {
			kv := strings.Split(param[i], ":")
			contextObj.StartParams[kv[0]] = kv[1]
		}

	}
	baizeCtx, baizeCtxCacelFun := context.WithCancel(context.Background())
	contextObj.BaizeCtx = baizeCtx
	contextObj.BaizeCtxCacelFun = baizeCtxCacelFun

	contextObj.IpfsPeerID = GetPeerID()
	contextObj.Events = make(map[string]*list.List)
	contextObj.Events[EVENT_START] = list.New()
	contextObj.Events[DYNAMIC_PARAM_CHANGE] = list.New()
	return nil

}

//IpfsStart ipfs启动完成
func IpfsStart(ipfsNode *core.IpfsNode) {
	contextObj.IpfsNode = ipfsNode
	contextObj.IpfsPeerID = ipfsNode.Identity.Pretty()

	contextObj.IsIpfsStart = true
	StartingLock.Unlock()
	StartingLockIsUnlocked = true
	//上报启动日志
	dataMap := make(map[string]string)
	dataMap[gather.EventActionKey] = "ipfs_daemon_start"
	gather.Gather(dataMap)

	//启动成功后记录标志文件
	writeStartedFlag()
	//触发启动事件
	TriggerEvent(EVENT_START)
	//执行block回收
	go gc.ConditionalGC(ipfsNode.Context(), ipfsNode, 0)

	fmt.Println(os.Stdout, time.Now().String()+" run in:"+runmodestat.GetCurrentRunMode())

}
func writeStartedFlag() {
	startedFlagPath := filepath.Join(contextObj.BaizeRootPath, "start-success")
	_, err := os.Stat(startedFlagPath)
	if os.IsNotExist(err) {
		file, err := os.Create(startedFlagPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error create started flag file: %s\n", err)
		} else {
			file.Close()
		}
	}
}

//TriggerEvent 触发事件
func TriggerEvent(evtName string) {
	for name, eventList := range contextObj.Events {
		if name == evtName {
			if eventList.Len() <= 0 {
				return
			}
			for element := eventList.Front(); element != nil; element = element.Next() {
				evtFunc := element.Value.(func())
				go evtFunc()
			}
		}
	}
}

//TriggerEvent 触发事件
func TriggerEventSync(evtName string) {
	for name, eventList := range contextObj.Events {
		if name == evtName {
			if eventList.Len() <= 0 {
				return
			}
			for element := eventList.Front(); element != nil; element = element.Next() {
				evtFunc := element.Value.(func())
				evtFunc()
			}
		}
	}
}

//ListenEvent 监听事件
func ListenEvent(evtName string, evtFun func()) {
	eventList := contextObj.Events[evtName]
	if nil == eventList {
		return
	}
	eventList.PushBack(evtFun)
}

//GetPeerID 获取节点id
func GetPeerID() string {
	if nil == contextObj {
		return ""
	}
	if "" != contextObj.IpfsPeerID {
		return contextObj.IpfsPeerID
	}
	ipfsCfgPath := filepath.Join(contextObj.RepoPath, "config")
	conf, err := serialize.Load(ipfsCfgPath)
	peerID := ""
	if err == nil {
		peerID = conf.Identity.PeerID
	}
	return peerID
}

//GetContext 返回上下文对象
func GetContext() *BaizeContext {
	return contextObj
}

//GetDynamicParamStr 获取动态参数字符串
func GetDynamicParamStr() string {
	dynamicParamStr := ""
	for k, v := range contextObj.DynamicParams {
		if "" == dynamicParamStr {
			dynamicParamStr = k + "\u0004" + v
		} else {
			dynamicParamStr = dynamicParamStr + "\u0005" + k + "\u0004" + v
		}
	}
	return dynamicParamStr
}

//GetStartParamStr 获取启动参数字符串
func GetStartParamStr() string {
	startParamStr := ""
	for k, v := range contextObj.StartParams {
		if "" == startParamStr {
			startParamStr = k + "\u0004" + v
		} else {
			startParamStr = startParamStr + "\u0005" + k + "\u0004" + v
		}
	}
	return startParamStr
}
func creatPathIfNotExist(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
	}
	if nil != err {
		return err
	}
	return nil

}

//TransTimeStr 转换为时间字符串，格式yyyy-MM-dd HH:mm:ss
func TransTimeStr(t time.Time) string {
	return strconv.Itoa(t.Year()) + "-" + strconv.Itoa(int(t.Month())) + "-" + strconv.Itoa(t.Day()) + " " + strconv.Itoa(t.Hour()) + ":" + strconv.Itoa(t.Minute()) + ":" + strconv.Itoa(t.Second())

}

//IncrMetricsCount 按key统计计数
func IncrMetricsCount(key string) {
	contextObj.MetricsCount[key] = contextObj.MetricsCount[key] + 1
}

func GetNetwork() string {
	startNetwork := contextObj.StartParams["network"]
	changeNetwork := contextObj.DynamicParams["network"]

	if "" == changeNetwork {
		return startNetwork
	}
	return changeNetwork
}
