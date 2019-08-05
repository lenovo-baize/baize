package reporter

import (
	"bytes"
	"container/list"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	context "baize/context"
)

func (dataReporter *DataReporter) sendToServer() {
	for {
		if dataReporter.isStop {
			fmt.Println("sendToServer stop")
			return
		}
		//如果没有发送的数据，睡眠5s
		if !dataReporter.hasSendFile() {
			time.Sleep(5 * time.Second)
		} else {
			dataReporter.doSend()
		}

	}

}
func (dataReporter *DataReporter) hasSendFile() bool {
	if cachedFilePaths.Len() > 0 {
		return true
	}
	return false
}

func (dataReporter *DataReporter) doSend() {
	pathList := dataReporter.getPathList()
	content := dataReporter.getSendContent(pathList)
	dataReporter.doSendWithRetry(content, pathList)
}

func (dataReporter *DataReporter) doSendWithRetry(content string, pathList *list.List) {
	content = context.GetContext().Version + "\u0003" + content
	//循环重试直到成功，如果失败，睡眠30秒后，在试
	sendErrNum := 0
	for {
		if dataReporter.isStop {
			return
		}
		//content http发送服务端
		sendResult := dataReporter.sendHTTP([]byte(content))
		//成功后处理
		if sendResult {
			dataReporter.removeFiles(pathList)
			return
		}
		sendErrNum = sendErrNum + 1
		context.IncrMetricsCount("report.send.http.error")
		//失败了睡30秒,重试
		sleepTime := 10 * time.Second
		switch {
		case 1 == sendErrNum:
			sleepTime = 10 * time.Second
		case 2 == sendErrNum:
			sleepTime = 20 * time.Second
		case 3 == sendErrNum:
			sleepTime = 40 * time.Second
		case 4 == sendErrNum:
			sleepTime = 80 * time.Second
		case sendErrNum >= 5:
			sleepTime = 160 * time.Second
		}
		time.Sleep(sleepTime)

	}
}
func (dataReporter *DataReporter) removeFiles(pathList *list.List) {
	for e := pathList.Front(); e != nil; e = e.Next() {
		path := e.Value.(string)
		delete(cachedFileContents, path)
		err := os.Remove(path)
		if nil != err {
			context.IncrMetricsCount("report.send.removeFile.error")
		}
		dataReporter.removeCachePath(path)
	}
}
func (dataReporter *DataReporter) getPathList() *list.List {
	batchNum := context.GetContext().Config.DataReport.GetMaxRecordNumOneTime()
	pathList := list.New()
	num := 0
	for element := cachedFilePaths.Front(); element != nil; element = element.Next() {
		if num <= batchNum {
			pathList.PushBack(element.Value)
		}
		num++
	}
	return pathList
}

func (dataReporter *DataReporter) getSendContent(pathList *list.List) string {
	records := ""
	for e := pathList.Front(); e != nil; e = e.Next() {
		path := e.Value.(string)
		record := dataReporter.getContentByPath(path)
		if "" == records {
			records = record
		} else {
			records = records + "\u0002" + record
		}

	}
	return records
}
func (dataReporter *DataReporter) getContentByPath(path string) string {
	data := cachedFileContents[path]
	if len(data) > 0 {
		return data
	}
	//如果缓存没有内容，从磁盘读
	fileContent, e := ioutil.ReadFile(path)
	if nil != e {
		context.IncrMetricsCount("report.send.readFile.error")
		return ""
	}
	//如果文件存在，但是內容为空，也丢弃掉
	if nil == fileContent || len(fileContent) <= 0 {
		return ""
	}
	return string(fileContent)
}
func (dataReporter *DataReporter) sendHTTP(content []byte) bool {
	if nil == content {
		return true
	}
	reportUrls := context.GetContext().Config.DataReport.ReportUrls
	// fmt.Println(string(content))
	for i := 0; i < len(reportUrls); i++ {
		reportURL := reportUrls[i]
		httpReq, err := http.NewRequest("POST", reportURL, bytes.NewReader(content))
		if nil != err {
			continue
		}
		httpReq.Header.Set("Connection", "Keep-Alive")
		httpRes, err := http.DefaultClient.Do(httpReq)
		if nil != err {
			continue
		}
		defer httpRes.Body.Close()
		status := httpRes.StatusCode
		if 200 != status {
			continue
		}
		return true
	}
	return false

}
