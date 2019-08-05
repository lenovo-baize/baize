package autoGather

import (
	datagathercfg "baize/config/datagatherCfg"
	"baize/context"
)

//PeriodParamKey 定时采集的定时配置项key
const PeriodParamKey = "Period"

//FieldsParamKey dynnamic中的fields字段配置
const FieldsParamKey = "Fields"

//DataGather 数据采集者接口
type DataGather interface {
	GetKey() string
	GetEventAction() string
	Start()
}

var constructMap = make(map[string]func(cfg *datagathercfg.DataGather) DataGather)

//RegConstruct 以key 注册采集者构造方法
func RegConstruct(gatherKey string, constructFun func(cfg *datagathercfg.DataGather) DataGather) {
	if nil != constructMap[gatherKey] {
		panic(gatherKey + " construct exist")
	}
	constructMap[gatherKey] = constructFun
}

var dataGatherMap = make(map[string]DataGather)

//RegDataGather 注册采集者对象
func RegDataGather(gatherKey string, eventAction string, dataGather DataGather) {
	dataGatherMap[gatherKey+"_"+eventAction] = dataGather
}

//SyncConfig 同步配置
func SyncConfig() {
	//只有新增的需要单独处理，取消配置的定时采集者会自动停止
	doNewCfgDataGather()
}
func removeDataGather(gatherKey string, eventAction string) {
	delete(dataGatherMap, gatherKey+"_"+eventAction)
}

//处理新增的配置
func doNewCfgDataGather() {
	dataGatherCfgs := baizectx.GetContext().Config.DataReport.DataGathers
	for i := 0; i < len(dataGatherCfgs); i++ {
		dataGatherCfg := dataGatherCfgs[i]
		dataGather := dataGatherMap[dataGatherCfg.Key+"_"+dataGatherCfg.EventAction]
		if nil == dataGather {
			initDataGather(&dataGatherCfg)
		}
	}
}
func initDataGather(dataGatherCfg *datagathercfg.DataGather) {
	constructFun := constructMap[dataGatherCfg.Key]
	if nil == constructFun {
		return
	}
	dataGather := constructFun(dataGatherCfg)
	dataGather.Start()
}

//InitAll 初始化所有的dataGather
func InitAll() {
	dataGatherCfgs := baizectx.GetContext().Config.DataReport.DataGathers
	if nil == dataGatherCfgs || len(dataGatherCfgs) <= 0 {
		return
	}
	for i := 0; i < len(dataGatherCfgs); i++ {
		dataGatherCfg := &dataGatherCfgs[i]
		initDataGather(dataGatherCfg)
	}
}
