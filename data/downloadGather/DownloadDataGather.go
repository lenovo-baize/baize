package downloadgather

import (
	"github.com/lenovo-baize/baize/runmode/runmodestat"
	"github.com/ipfs/go-cid"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	bzCtx "github.com/lenovo-baize/baize/context"
	"github.com/lenovo-baize/baize/data/gather"
	"github.com/lenovo-baize/baize/runmode/runmodedetect"
)

//DownloadData 下载打桩数据
type DownloadData struct {
	DownloadID        string
	DataSrc           string
	DownloadURL       string
	DownloadChannel   string
	Cid               cid.Cid
	DownloadURLParams string
	ReqRange          string
	isLocalHas        string
	StartTime         time.Time
	Err               error
	TotalSize         int64
	RespSize          int64
	FileType          string
}

//DownloadExtHandler 下载扩展处理
func DownloadExtHandler(w http.ResponseWriter, r *http.Request) *DownloadData {
	params := r.URL.Query()

	defaultDownloadHeader := bzCtx.GetContext().Config.DownloadConfig.DownloadHeader["default"]
	for k, v := range defaultDownloadHeader {
		w.Header().Set(k, v)
	}

	channel := params.Get("channel")
	channelDownloadHeader := bzCtx.GetContext().Config.DownloadConfig.DownloadHeader[channel]
	for k, v := range channelDownloadHeader {
		w.Header().Set(k, v)
	}
	ftype := params.Get("ftype")
	fTypeDownloadHeader := bzCtx.GetContext().Config.DownloadConfig.DownloadHeader[ftype]
	for k, v := range fTypeDownloadHeader {
		w.Header().Set(k, v)
	}
	downloadData := NewDownloadData()
	downloadData.StartTime = time.Now()
	downloadData.DataSrc = "gateway"
	downloadData.DownloadURL = r.URL.Path
	downloadData.DownloadURLParams = r.URL.RawQuery
	downloadData.DownloadChannel = channel
	downloadData.ReqRange = r.Header.Get("Range")
	return downloadData
}

var downLock sync.Mutex
var downSeq int
var bootStrapProc io.Closer

//NewDownloadData 构建下载数据对象
func NewDownloadData() *DownloadData {
	downLock.Lock()
	defer downLock.Unlock()
	runmodestat.DownNum = runmodestat.DownNum + 1
	downSeq++
	downloadData := &DownloadData{}
	downloadData.DownloadID = bzCtx.GetContext().IpfsPeerID + "_" + strconv.Itoa(time.Now().Nanosecond()) + "_" + strconv.Itoa(downSeq)
	return downloadData
}

//StartDownload 开始下载日志上报
func StartDownload(data *DownloadData) {
	dataMap := make(map[string]string)
	dataMap[gather.EventActionKey] = "D_sD"
	dataMap["download_id"] = data.DownloadID
	dataMap["data_src"] = data.DataSrc
	dataMap["file_type"] = data.FileType
	dataMap["download_url"] = data.DownloadURL
	dataMap["download_url_params"] = data.DownloadURLParams
	dataMap["download_channel"] = data.DownloadChannel
	dataMap["req_range"] = data.ReqRange
	dataMap["downNum"] = strconv.Itoa(runmodestat.DownNum)
	trafficData := rummodedetect.GetAndroidPhoneTrafficData()
	if nil != trafficData {
		dataMap["mobile_in"] = trafficData["mobile_in"]
		dataMap["mobile_out"] = trafficData["mobile_out"]
		dataMap["wifi_in"] = trafficData["wifi_in"]
		dataMap["wifi_out"] = trafficData["wifi_out"]
		dataMap["trafficerr"] = trafficData["trafficerr"]
	}
	dataMap["conns"] = strconv.Itoa(len(bzCtx.GetContext().IpfsNode.PeerHost.Network().Conns()))

	gather.Gather(dataMap)
}

//CompletDownload 下载完成数据采集
func CompletDownload(data *DownloadData) {
	downLock.Lock()
	defer downLock.Unlock()
	runmodestat.DownNum = runmodestat.DownNum - 1
	dataMap := make(map[string]string)
	dataMap[gather.EventActionKey] = "D_eD"
	dataMap["download_id"] = data.DownloadID
	dataMap["data_src"] = data.DataSrc
	dataMap["file_type"] = data.FileType
	dataMap["download_url"] = data.DownloadURL
	dataMap["download_url_params"] = data.DownloadURLParams
	dataMap["download_channel"] = data.DownloadChannel
	dataMap["cid"] = data.Cid.String()
	dataMap["req_range"] = data.ReqRange
	dataMap["time"] = strconv.FormatInt(time.Since(data.StartTime).Nanoseconds(), 10)
	dataMap["total_size"] = strconv.FormatInt(data.TotalSize, 10)
	dataMap["resp_size"] = strconv.FormatInt(data.RespSize, 10)
	dataMap["is_local_has"] = data.isLocalHas
	dataMap["downNum"] = strconv.Itoa(runmodestat.DownNum)
	dataMap["conns"] = strconv.Itoa(len(bzCtx.GetContext().IpfsNode.PeerHost.Network().Conns()))

	errStr := ""
	if nil != data.Err {
		errStr = data.Err.Error()
	}
	dataMap["error"] = errStr
	trafficData := rummodedetect.GetAndroidPhoneTrafficData()
	if nil != trafficData {
		dataMap["mobile_in"] = trafficData["mobile_in"]
		dataMap["mobile_out"] = trafficData["mobile_out"]
		dataMap["wifi_in"] = trafficData["wifi_in"]
		dataMap["wifi_out"] = trafficData["wifi_out"]
		dataMap["trafficerr"] = trafficData["trafficerr"]
	}
	gather.Gather(dataMap)
}

//SetBeforeDowloadInfo 设置下载前的信息
func SetBeforeDowloadInfo(data *DownloadData) {
	//本地是否有
	isLocalHasFile, err := bzCtx.GetContext().IpfsNode.Blockstore.Has(data.Cid)
	if nil != err {
		data.isLocalHas = "err:" + err.Error()
	} else if isLocalHasFile {
		data.isLocalHas = "true"
	} else {
		data.isLocalHas = "false"
	}

}
