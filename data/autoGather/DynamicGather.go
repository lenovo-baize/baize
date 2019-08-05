package autoGather

import (
	"fmt"
	"baize/runmode/runmodestat"
	"os"
	"strconv"
	"strings"
	"time"

	"baize/config/datagatherCfg"
	context "baize/context"
	"baize/data/gather"
	"baize/runmode/runmodedetect"
)

//DynamicGatherKey 事件名称
const DynamicGatherKey = "dynamic"

//DynamicGather 采集者
type DynamicGather struct {
	Key         string
	EventAction string
	Param       map[string]string
}

func init() {
	//注册构造函数
	RegConstruct(DynamicGatherKey, newDynamicDataGather)
}

//构建指标采集者
func newDynamicDataGather(cfg *datagathercfg.DataGather) DataGather {
	dynamicDataGather := &DynamicGather{
		Key:         DynamicGatherKey,
		EventAction: cfg.EventAction,
		Param:       cfg.Param,
	}
	dataGather := interface{}(dynamicDataGather).(DataGather)
	//注册采集者对象
	RegDataGather(DynamicGatherKey, dynamicDataGather.EventAction, dataGather)
	//启动成功上报1次
	context.ListenEvent(context.EVENT_START, dynamicDataGather.gather)

	return dataGather
}

//GetKey 获取key
func (dataGather *DynamicGather) GetKey() string {
	return dataGather.Key
}

//GetEventAction 获取eventaction
func (dataGather *DynamicGather) GetEventAction() string {
	return dataGather.EventAction
}

//Start 启动采集者，定时执行采集，
func (dataGather *DynamicGather) Start() {
	go dataGather.execute()
}
func (dataGather *DynamicGather) execute() {
	fmt.Println("DynamicGather-" + dataGather.EventAction + " doGather start")
	nextPeriod := dataGather.getPeriod()
	if 0 == nextPeriod {
		removeDataGather(dataGather.Key, dataGather.EventAction)
		fmt.Println("DynamicGather-" + dataGather.EventAction + " doGather period is 0 not start")
		return
	}

	timer := time.NewTimer(nextPeriod)
	for {
		select {
		case <-timer.C:
			dataGather.gather()
			nextPeriod := dataGather.getPeriod()
			if 0 == nextPeriod {
				timer.Stop()
				removeDataGather(dataGather.Key, dataGather.EventAction)
				fmt.Println("DynamicGather-" + dataGather.EventAction + " doGather period is 0 end")
				return
			}
			timer.Reset(nextPeriod)
		case <-context.GetContext().BaizeCtx.Done():
			timer.Stop()
			removeDataGather(dataGather.Key, dataGather.EventAction)
			fmt.Println("DynamicGather-" + dataGather.EventAction + " doGather stop")
			return
		}
	}

}
func (dataGather *DynamicGather) getPeriod() time.Duration {
	dataGatherCfg := context.GetContext().Config.DataReport.GetDataGather(dataGather.Key + "_" + dataGather.EventAction)
	if nil == dataGatherCfg {
		return 0
	}
	period, err := time.ParseDuration(dataGatherCfg.Param[PeriodParamKey])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parse time: %s err:%s\n", dataGatherCfg.Param[PeriodParamKey], err)
		period = 600 * time.Second
	}
	return period
}
func (dataGather *DynamicGather) gather() {
	gather.Gather(dataGather.buildDynamicData())
}
func (dataGather *DynamicGather) buildDynamicData() map[string]string {
	startTime := time.Now()
	data := dataGather.getData()
	if !context.GetContext().IsIpfsStart {
		return data
	}
	if nil == data {
		return nil
	}
	data["collect_time"] = strconv.FormatInt(time.Since(startTime).Nanoseconds(), 10)
	data[gather.EventActionKey] = dataGather.EventAction
	return data

}
func (dataGather *DynamicGather) getData() map[string]string {
	filterData := make(map[string]string)
	//client模式，只上报流量信息
	if runmodestat.GetCurrentRunMode() == runmodestat.RUN_MODE_CLIENT {
		pickupAndoridPhoneTrafficData(filterData)
		connNumData := BuildConnNumsData()
		for key, value := range connNumData {
			filterData[key] = value
		}
		return filterData
	}
	fields := dataGather.Param[FieldsParamKey]
	if "" == fields || len(fields) <= 0 {
		return nil
	}
	allData := BuildIpfsMetricsData()
	connNumData := BuildConnNumsData()
	for key, value := range connNumData {
		allData[key] = value
	}
	baizeCountNumData := context.GetContext().MetricsCount
	for key, value := range baizeCountNumData {
		allData[key] = strconv.Itoa(value)
	}
	if "all" == fields {
		return allData
	}
	fieldsArray := strings.Split(fields, ",")
	for i := 0; i < len(fieldsArray); i++ {
		key := fieldsArray[i]
		if key == "andorid_phone_traffic" {
			pickupAndoridPhoneTrafficData(filterData)
		} else {
			filterData[key] = allData[key]
		}

	}

	return filterData
}

func pickupAndoridPhoneTrafficData(data map[string]string) {
	trafficData := rummodedetect.GetAndroidPhoneTrafficData()
	if nil != trafficData {
		data["mobile_in"] = trafficData["mobile_in"]
		data["mobile_out"] = trafficData["mobile_out"]
		data["wifi_in"] = trafficData["wifi_in"]
		data["wifi_out"] = trafficData["wifi_out"]
		data["trafficerr"] = trafficData["trafficerr"]
	}
}
