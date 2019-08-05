package rummodedetect

import (
	"github.com/lenovo-baize/baize/context"
	"github.com/lenovo-baize/baize/data/gather"
	"github.com/lenovo-baize/baize/runmode/runmodestat"
	"fmt"
	"os"
	"sync"
	"time"
)

//DetectRunModeBeforeStart 启动ipfs前检测运行模式
func DetectRunModeBeforeStart() {
	detectMethod := getDetectMethod()
	runDetectMethod(detectMethod)
	go Time2DetactDownload()
}
func getDetectMethod() string {
	source := baizectx.GetContext().StartParams["source"]
	runModeDetectConfig := baizectx.GetContext().Config.IpfsConfig.RunModeDetectConfig[source]
	return runModeDetectConfig.DetectMethod
}
func runDetectMethod(detectMethod string) {
	preNetwork = baizectx.GetNetwork()
	//当前只有安卓手机要检测，其他直接分享模式启动
	switch detectMethod {
	case "android-phone":
		//检测在安卓手机设备上的运行模式
		DetectRunModeOnAndroidPhone()
		return
	default:
		runmodestat.SetModeConfig(baizectx.GetContext().Config.IpfsConfig.ModeConfig)
		runmodestat.SetCurrentRunMode(runmodestat.RUN_MODE_SHARE)

	}
}

var changeModLock sync.Mutex

func changeToShareMode(reason string) {
	defer reportRunModeChange(runmodestat.GetCurrentRunMode(), runmodestat.RUN_MODE_SHARE, reason)
	runmodestat.SetModeConfig(baizectx.GetContext().Config.IpfsConfig.ModeConfig)
	runmodestat.SetCurrentRunMode(runmodestat.RUN_MODE_SHARE)

	if baizectx.GetContext().IsIpfsStart {
		changeModLock.Lock()
		defer changeModLock.Unlock()
		baizectx.GetContext().IpfsNode.PeerHost.Network().Close()
		fmt.Println(os.Stdout, time.Now().String()+"changeToShareMode trigger runmode change")
		runmodestat.TriggerEventSync(runmodestat.EVENT_RUNMODE_CHANGE)
		CloseBootStrap()
		fmt.Println(os.Stdout, time.Now().String()+"changeToShareMode CloseAllConns")
		CloseAllConns()
		fmt.Println(os.Stdout, time.Now().String()+"changeToShareMode Bootstrap")
		//baizectx.GetContext().IpfsNode.ConnectBootstrapPeers(baizectx.GetContext().IpfsCtx, core.DefaultBootstrapConfig)
		//baizectx.GetContext().IpfsNode.Bootstrap(core.DefaultBootstrapConfig)
	}

}
func changeToClientMode(reason string) {
	defer reportRunModeChange(runmodestat.GetCurrentRunMode(), runmodestat.RUN_MODE_CLIENT, reason)
	runmodestat.SetModeConfig(baizectx.GetContext().Config.IpfsConfig.ModeConfig)
	runmodestat.SetCurrentRunMode(runmodestat.RUN_MODE_CLIENT)
	if baizectx.GetContext().IsIpfsStart {
		changeModLock.Lock()
		defer changeModLock.Unlock()
		fmt.Println(os.Stdout, time.Now().String()+"changeToClientMode trigger runmode change")
		runmodestat.TriggerEventSync(runmodestat.EVENT_RUNMODE_CHANGE)
		fmt.Println(os.Stdout, time.Now().String()+"changeToClientMode CloseBootStrap")
		CloseBootStrap()
		fmt.Println(os.Stdout, time.Now().String()+"changeToClientMode CloseAllConns")
		CloseAllConns()
		//baizectx.GetContext().IpfsNode.ConnectBootstrapPeers(baizectx.GetContext().IpfsCtx, core.DefaultBootstrapConfig)
	}
}

func Time2DetactDownload() {
	timer := time.NewTimer(500 * time.Millisecond)
	for {
		select {
		case <-timer.C:
			downloadPorcess()
			timer.Reset(500 * time.Millisecond)
		case <-baizectx.GetContext().BaizeCtx.Done():
			timer.Stop()
			fmt.Println("Time2DetactDownload stop")
			return
		}
	}
}
func downloadPorcess() {
	if !baizectx.GetContext().IsIpfsStart {
		return
	}
	changeModLock.Lock()
	defer changeModLock.Unlock()
	if runmodestat.DownNum > 0 {
		//baizectx.GetContext().IpfsNode.ConnectBootstrapPeers(baizectx.GetContext().IpfsCtx, core.DefaultBootstrapConfig)
		//if nil == baizectx.GetContext().IpfsNode.Bootstrapper && runmodestat.GetCurrentRunMode() == runmodestat.RUN_MODE_CLIENT {
		//	baizectx.GetContext().IpfsNode.Bootstrap(core.DefaultBootstrapConfig)
		//}
		return
	}

	if runmodestat.GetCurrentRunMode() == runmodestat.RUN_MODE_CLIENT {
		CloseBootStrap()
		runmodestat.BootstrapTime++
		//if runmodestat.BootstrapTime > 240 {
		CloseAllConns()
		runmodestat.BootstrapTime = 0
		//}
	}
}
func CloseAllConns() {
	conns := baizectx.GetContext().IpfsNode.PeerHost.Network().Conns()
	if len(conns) > 0 {
		for _, conn := range conns {
			conn.Close()
		}
	}

}
func CloseBootStrap() {
	if nil != baizectx.GetContext().IpfsNode && nil != baizectx.GetContext().IpfsNode.Bootstrapper {
		baizectx.GetContext().IpfsNode.Bootstrapper.Close()
		baizectx.GetContext().IpfsNode.Bootstrapper = nil
	}
}
func reportRunModeChange(source string, target string, reason string) {
	dataMap := make(map[string]string)
	dataMap[gather.EventActionKey] = "change_run_mode"
	dataMap["current"] = source
	dataMap["target"] = target
	dataMap["reason"] = reason
	gather.Gather(dataMap)
}

func SyncConfig() {
	runmodestat.SetModeConfig(baizectx.GetContext().Config.IpfsConfig.ModeConfig)
}

func GetCurrentRunMode() string {
	return runmodestat.GetCurrentRunMode()
}
