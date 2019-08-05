package datagathercfg

//DataReport 数据上报配置
type DataReport struct {
	ReportUrls          []string
	MaxRecordNumOneTime int
	MaxCacheFileSize    int64
	DataGathers         []DataGather
	DisableEventActions []string
}

//GetMaxCacheFileSize 获取本地磁盘缓存待上报文件的最大大小
func (datareport *DataReport) GetMaxCacheFileSize() int64 {
	//默认5M
	if datareport.MaxCacheFileSize <= 0 {
		return 5242880
	}
	return datareport.MaxCacheFileSize
}

//GetMaxRecordNumOneTime 获取上报时，每次最大记录数
func (datareport *DataReport) GetMaxRecordNumOneTime() int {
	//默认按最大100条一个批次，批量上报数据
	if datareport.MaxRecordNumOneTime <= 0 {
		return 100
	}
	return datareport.MaxRecordNumOneTime
}

//IsDisableEventAction 获取指定事件是否取消采集
func (datareport *DataReport) IsDisableEventAction(eventAction string) bool {
	for i := 0; i < len(datareport.DisableEventActions); i++ {
		if datareport.DisableEventActions[i] == eventAction {
			return true
		}
	}
	return false
}

//GetDataGather 根据事件名获取对应的采集配置
func (datareport *DataReport) GetDataGather(key string) *DataGather {
	dataGathers := datareport.DataGathers
	for i := 0; i < len(dataGathers); i++ {
		dataGather := dataGathers[i]
		if (dataGather.Key + "_" + dataGather.EventAction) == key {
			return &dataGather
		}
	}
	return nil
}
