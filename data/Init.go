package data

import (
	// "fmt"

	context "baize/context"
	"baize/data/autoGather"
	"baize/data/gather"
	"baize/data/reporter"
)

//Init 初始化数据上报模块
func Init() error {
	//启动上报者
	dataReporter, err := reporter.New()
	if nil != err {
		return err
	}
	dataReporter.Start()
	//启动采集者
	autoGather.InitAll()
	//监听退出事件，关闭采集数据管道
	go func() {
		<-context.GetContext().BaizeCtx.Done()
		gather.IsReportStarted = false
		close(gather.ReportDataChan)
	}()
	return nil

}
