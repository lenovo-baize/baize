package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	ipfs "github.com/ipfs/go-ipfs"
	context "github.com/lenovo-baize/baize/context"
	"github.com/lenovo-baize/baize/data/gather"
	"github.com/lenovo-baize/baize/runmode/runmodestat"
)

func (dataReporter *DataReporter) reviceData() {
	dataChan := gather.ReportDataChan
	for {
		select {
		case data := <-dataChan:
			//是否被禁用
			if !isDisabled(data) {
				dataReporter.doRecive(transLogFormat(data))
			}

		case <-context.GetContext().BaizeCtx.Done():
			close(dataChan)
			dataReporter.isStop = true
			fmt.Println("reviceData stop")
			return
		}
	}

}
func isDisabled(data map[string]string) bool {
	eventAction := data[gather.EventActionKey]
	disableEventActions := context.GetContext().Config.DataReport.DisableEventActions
	if nil != disableEventActions && len(disableEventActions) > 0 {
		if context.GetContext().Config.DataReport.IsDisableEventAction(eventAction) {
			return true
		}
	}
	return false
}
func transLogFormat(data map[string]string) string {
	//补全一些基础信息，时间地址等
	recordStr := data[gather.EventActionKey] + "\u0001"
	recordStr += context.GetPeerID() + "\u0001"
	recordStr += context.GetContext().Version + "\u0001"
	recordStr += ipfs.CurrentVersionNumber + "\u0001"
	recordStr += runtime.GOARCH + "\u0001"
	recordStr += runtime.GOOS + "\u0001"
	now := time.Now()
	zoneName, offset := now.Zone()
	recordStr += context.TransTimeStr(now) + "\u0001"
	recordStr += zoneName + "|" + strconv.Itoa(offset) + "\u0001"
	recordStr += context.GetStartParamStr() + "\u0001"
	recordStr += context.GetDynamicParamStr() + "\u0001"
	recordStr += runmodestat.GetCurrentRunMode() + "\u0001"
	delete(data, gather.EventActionKey)
	bizDataStr := ""
	for k, v := range data {
		if "" == bizDataStr {
			bizDataStr = k + "\u0004" + v
		} else {
			bizDataStr = bizDataStr + "\u0005" + k + "\u0004" + v
		}
	}
	recordStr += bizDataStr

	return recordStr
}
func (dataReporter *DataReporter) doRecive(data string) {
	filePath := dataReporter.writeDataToFile(data)
	if "" == filePath {
		return
	}
	cachedFilePaths.PushBack(filePath)
	//缓存中只缓存50个文件内容，超过的，从磁盘读取
	if len(cachedFileContents) <= 50 {
		cachedFileContents[filePath] = data
	}
}
func (dataReporter *DataReporter) writeDataToFile(data string) string {
	//创建文件，并将数据写入
	file := dataReporter.creatCacheFile()
	if nil == file {
		return ""
	}
	defer file.Close()

	_, err := file.Write([]byte(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error write  data file: %s\n", err)
		//写失败了，将文件删除
		os.Remove(file.Name())
		context.IncrMetricsCount("report.recive.writeFile.error")
		return ""
	}
	return file.Name()
}
func (dataReporter *DataReporter) creatCacheFile() *os.File {
	fileName := strconv.FormatInt(time.Now().UnixNano(), 10)
	filePath := filepath.Join(context.GetContext().DataGatherTmpPath, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error create data file: %s\n", err)
		context.IncrMetricsCount("report.recive.createFile.error")
		return nil
	}
	return file

}
