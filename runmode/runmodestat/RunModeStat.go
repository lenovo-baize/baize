package runmodestat

import (
	"container/list"
	"time"

	"github.com/lenovo-baize/baize/config/ipfsCfg"
)

var RUN_MODE_SHARE = "share"
var RUN_MODE_CLIENT = "client"
var DownNum = 0
var BootstrapTime = 0
var IsBootstrapNode = false

const EVENT_RUNMODE_CHANGE = "runmodeChange"

var events = make(map[string]*list.List)

func init() {
	events[EVENT_RUNMODE_CHANGE] = list.New()
}

//运行模式改变事件

var currentRunMode string
var modeConfig map[string]ipfscfg.RunModeConfig

func SetCurrentRunMode(runMode string) {
	currentRunMode = runMode
}
func GetCurrentRunMode() string {
	return currentRunMode
}

func SetModeConfig(cfg map[string]ipfscfg.RunModeConfig) {
	modeConfig = cfg
}
func GetCurrentRunModeConfig() ipfscfg.RunModeConfig {
	return modeConfig[currentRunMode]
}

//判断并等待直到share模式返回
func WaitShareMode() {
	for {
		if currentRunMode == RUN_MODE_SHARE {
			return
		}
		time.Sleep(40 * time.Second)
	}
}

//ListenEvent 监听事件
func ListenEvent(evtName string, evtFun func()) {
	eventList := events[evtName]
	if nil == eventList {
		return
	}
	eventList.PushBack(evtFun)
}

//TriggerEvent 触发事件
func TriggerEvent(evtName string) {
	for name, eventList := range events {
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
	for name, eventList := range events {
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
