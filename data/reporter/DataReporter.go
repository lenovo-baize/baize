package reporter

import (
	"container/list"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	context "baize/context"
	"baize/data/gather"
)

// DataReporter 数据上报者
type DataReporter struct {
	sync.Mutex
	isStop bool
}

//New 数据上报者构建方法
func New() (*DataReporter, error) {
	dataReporter := creatDataReporter()
	//读取磁盘缓存数据目录下的所有文件，准备发送
	dataReporter.getAllDataFliePath()
	return dataReporter, nil
}
func creatDataReporter() *DataReporter {
	dataReporter := &DataReporter{
		isStop: false,
	}
	return dataReporter
}

//缓存的文件内容
var cachedFileContents = make(map[string]string)

//缓存的文件路径
var cachedFilePaths = list.New()

//Start 启动数据上报
func (dataReporter *DataReporter) Start() {
	//启动从采集者接收数据
	go dataReporter.reviceData()
	//启动将接收的数据发送到服务端
	go dataReporter.sendToServer()
	//启动监控缓存文件大小，超过阀值进行删除
	go dataReporter.watchDiskCacheSize()

	gather.IsReportStarted = true
}

func (dataReporter *DataReporter) getAllDataFliePath() {
	entries, err := ioutil.ReadDir(context.GetContext().DataGatherTmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read data path error: %s\n", err)
		return
	}
	for _, entry := range entries {
		cachedFilePaths.PushBack(filepath.Join(context.GetContext().DataGatherTmpPath, entry.Name()))
	}
}

func (dataReporter *DataReporter) watchDiskCacheSize() {
	//启动后先执行1次
	dataReporter.gcCacheFiles()
	//后续5分钟执行1次
	period := 300 * time.Second
	timer := time.NewTimer(period)
	for {
		select {
		case <-timer.C:
			dataReporter.gcCacheFiles()
			timer.Reset(period)
		case <-context.GetContext().BaizeCtx.Done():
			timer.Stop()
			dataReporter.isStop = true
			fmt.Println("watchCacheSize stop")
			return
		}
	}
}
func (dataReporter *DataReporter) gcCacheFiles() {
	thresholdSize := context.GetContext().Config.DataReport.GetMaxCacheFileSize()
	fileInfos := dataReporter.getDirFileInfos()
	if nil == fileInfos {
		return
	}
	totalFileSize := dataReporter.getTotalCacheFileSize(fileInfos)
	if totalFileSize < thresholdSize {
		return
	}
	dataReporter.doClean(fileInfos, totalFileSize, thresholdSize)
	//由于这边删除的数据，数据还在继续生成，上面删到阀值90%以下后，这里在检测1次，如果又写超了继续删
	dataReporter.gcCacheFiles()

}
func (dataReporter *DataReporter) doClean(fileInfos []os.FileInfo, totalSize int64, thresholdSize int64) {
	cleanNum := 0
	cleanSize := int64(0)
	maxSize := float64(thresholdSize) * float64(0.9)
	for _, entry := range fileInfos {
		path := filepath.Join(context.GetContext().DataGatherTmpPath, entry.Name())

		cleanSize = cleanSize + entry.Size()
		cleanNum++
		dataReporter.removeFile(path)
		curSize := totalSize - cleanSize
		//清除到只有阀值的90%时停止
		if float64(curSize) < maxSize {
			break
		}

		if dataReporter.isStop {
			return
		}

	}
	fmt.Printf("clean file num:s%,size:s%", cleanNum, cleanSize)
}
func (dataReporter *DataReporter) removeFile(path string) {
	err := os.Remove(path)
	if nil != err {
		context.IncrMetricsCount("report.removeFile.error")
	}
	//文件删除后，内存缓存的也删除掉
	delete(cachedFileContents, path)
	dataReporter.removeCachePath(path)
}
func (dataReporter *DataReporter) removeCachePath(path string) {
	pathElement := dataReporter.getCachePathElement(path)
	if nil != pathElement {
		cachedFilePaths.Remove(pathElement)
	}
}
func (dataReporter *DataReporter) getCachePathElement(path string) *list.Element {
	for e := cachedFilePaths.Front(); e != nil; e = e.Next() {
		if e.Value == path {
			return e
		}
	}
	return nil
}
func (dataReporter *DataReporter) getDirFileInfos() []os.FileInfo {
	entries, err := ioutil.ReadDir(context.GetContext().DataGatherTmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read data path error: %s\n", err)
		return nil
	}
	return entries
}

//获取总文件大小
func (dataReporter *DataReporter) getTotalCacheFileSize(fileInfos []os.FileInfo) int64 {
	fileSize := int64(0)
	for _, entry := range fileInfos {
		fileSize = fileSize + entry.Size()
	}
	return fileSize
}
