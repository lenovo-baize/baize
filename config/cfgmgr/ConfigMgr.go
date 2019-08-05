package cfgmgr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	serialize "github.com/ipfs/go-ipfs-config/serialize"

	config "baize/config"
	datagathercfg "baize/config/datagatherCfg"
	download "baize/config/downloadCfg"
	ipfscfg "baize/config/ipfsCfg"
	context "baize/context"
	"baize/data/autoGather"
	"baize/runmode/runmodedetect"
	"baize/runmode/runmodestat"
)

var syncConfigLock sync.Mutex

//InitConfig 初始化配置对象
func InitConfig() error {
	if nil != context.GetContext().Config {
		return nil
	}
	if !isConfigFileExist() {
		err := genDefaultConfigFile()
		if err != nil {
			return err
		}
	}
	config, err := readConfig()
	if nil != err {
		return err
	}
	context.GetContext().Config = config
	return nil
}

func genDefaultConfigFile() error {
	ctx := context.GetContext()
	os.Remove(ctx.BaizeConfigPath)

	file, err := os.Create(ctx.BaizeConfigPath)
	if nil != err {
		fmt.Fprintf(os.Stderr, "create default baize config file error: %s\n", err)
		return err
	}
	defer file.Close()
	defaultConfig, err := genDefaultConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen default config json str error: %s\n", err)
		return err
	}
	_, err = file.Write(defaultConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "save default baize config file error: %s\n", err)
		return err
	}
	return nil
}
func genDefaultConfig() ([]byte, error) {
	cfg := &config.Config{}
	cfg.Version = context.GetContext().Version
	cfg.CfgUpdatePeriod = "10m"
	dataReportConf := datagathercfg.DataReport{}
	dataReportConf.ReportUrls = []string{"http://ipfsdata.lenovomm.cn/peer/report", "http://52.80.252.8:8888/peer/report", "http://54.223.108.72:8888/peer/report", "http://52.80.179.53:8888/peer/report"}
	// dataReportConf.ReportUrls = []string{}
	dataReportConf.MaxRecordNumOneTime = 100
	dataReportConf.MaxCacheFileSize = 20971520
	dataGather := datagathercfg.DataGather{}
	dataGather.Key = "dynamic"
	dataGather.EventAction = "ipfs_default_data"
	dataGather.Param = make(map[string]string)
	dataGather.Param["Fields"] = "storage_usage,ipfs_bitswap_recv_all_blocks_bytes_sample_sum,ipfs_bitswap_recv_all_blocks_bytes_sample_count,ipfs_bitswap_recv_dup_blocks_bytes_sample_sum,ipfs_bitswap_recv_dup_blocks_bytes_sample_count,ipfs_bitswap_sent_all_blocks_bytes_sample_sum,ipfs_bitswap_sent_all_blocks_bytes_sample_count,ipfs_bitswap_wantlist_total,ipfs_p2p_peers_total,band_width,andorid_phone_traffic"
	// dataGather.Param["Fields"] = "band_width_total,band_width_/ipfs/kad/1.0.0,band_width_/libp2p/circuit/relay/0.1.0,band_width_/ipfs/id/1.0.0"
	dataGather.Param["Period"] = "1m"
	dataReportConf.DataGathers = []datagathercfg.DataGather{dataGather}
	dataReportConf.DisableEventActions = []string{}
	cfg.DataReport = dataReportConf

	ipfsCfg := ipfscfg.IpfsConfig{}
	ipfsCfg.Bootstrap = []string{"/dns4/ipfs.lenovomm.cn/tcp/4001/ipfs/QmZChMXdJQ3z9hVSBMQB3xjJqMBsin83QFfR7D84Snh3Yk", "/ip4/52.80.252.8/tcp/4001/ipfs/QmRzZgXj4q6qgxYHQHCDyTY1iE3tiZh6PLeTxv4PD7b8fd", "/ip4/54.223.108.72/tcp/4001/ipfs/QmZChMXdJQ3z9hVSBMQB3xjJqMBsin83QFfR7D84Snh3Yk", "/ip4/52.80.179.53/tcp/4001/ipfs/QmXAUxBMJWwbLbwSNTPEJdPA8mHt4rk8v34UTHQ9DosmWE"}
	//ipfsCfg.Bootstrap = []string{"/ip4/10.109.4.71/tcp/4001/ipfs/QmP3dQD9eNd4b78VgULEUcYVtzUb8KWKRutSXipaRhTXbX"}
	ipfsCfg.StorageMax = "200M"
	ipfsCfg.StorageGCWatermark = 90
	ipfsCfg.GCPeriod = "12h"
	ipfsCfg.ModeConfig = make(map[string]ipfscfg.RunModeConfig)
	shareRunModeCfg := ipfscfg.RunModeConfig{}
	shareRunModeCfg.DhtClient = false
	shareRunModeCfg.BitSwapClient = false
	shareRunModeCfg.EnableRelayHop = true
	shareRunModeCfg.EnableRelayAddr = true
	shareRunModeCfg.EnableDownLoad = true
	shareRunModeCfg.DisableDownLoadRespCode = 503
	shareRunModeCfg.MinConn = 600
	shareRunModeCfg.MaxConn = 900
	shareRunModeCfg.LimitRefreshRoutingTable = false
	ipfsCfg.ModeConfig[runmodestat.RUN_MODE_SHARE] = shareRunModeCfg

	clientRunModeCfg := ipfscfg.RunModeConfig{}
	clientRunModeCfg.DhtClient = true
	clientRunModeCfg.BitSwapClient = true
	clientRunModeCfg.EnableRelayHop = false
	clientRunModeCfg.EnableRelayAddr = false
	clientRunModeCfg.EnableDownLoad = true
	clientRunModeCfg.DisableDownLoadRespCode = 503
	clientRunModeCfg.MinConn = 600
	clientRunModeCfg.MaxConn = 900
	clientRunModeCfg.LimitRoutTableNum = 30
	clientRunModeCfg.LimitRefreshRoutingTable = true
	ipfsCfg.ModeConfig[runmodestat.RUN_MODE_CLIENT] = clientRunModeCfg

	ipfsCfg.RunModeDetectConfig = make(map[string]ipfscfg.RunModeDetectConfig)

	androidPhoneRunModeDetectConfig := ipfscfg.RunModeDetectConfig{}
	androidPhoneRunModeDetectConfig.DetectMethod = "android-phone"
	androidPhoneRunModeDetectConfig.DetectPeriod = "30s"
	ipfsCfg.RunModeDetectConfig["android-phone"] = androidPhoneRunModeDetectConfig

	ipfsCfg.MobileUseThreshold = 1 * 1024
	ipfsCfg.MobileUseNumThreshold = 5
	cfg.IpfsConfig = ipfsCfg

	downloadCfg := download.DownloadConfig{}
	downloadCfg.DownloadHeader = make(map[string]map[string]string)
	downloadCfg.DownloadHeader["apk"] = make(map[string]string)
	downloadCfg.DownloadHeader["apk"]["Content-Type"] = "application/vnd.android.package-archive"
	cfg.DownloadConfig = downloadCfg

	return json.Marshal(cfg)
}

//读取配置文件
func readConfig() (*config.Config, error) {
	path := context.GetContext().BaizeConfigPath
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg config.Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	if cfg.Version == context.GetContext().Version {
		return &cfg, nil
	}

	return nil, errors.New("local config file version not match baizContext version")
}
func isConfigFileExist() bool {
	path := context.GetContext().BaizeConfigPath
	_, err := os.Lstat(path)
	if nil != err {
		return false
	}
	return true
}

//SyncServerConfig 同步服务端的配置
func SyncServerConfig() error {
	ctx := context.GetContext()
	syncConfigLock.Lock()
	defer syncConfigLock.Unlock()
	config, e := getServerConfigWithRetry()
	if e != nil {
		fmt.Fprintf(os.Stderr, "error get config from server: %s\n", e)
		return e
	}

	if config.Version != ctx.Version {
		fmt.Fprintf(os.Stderr, "server return config file version not match baizContext version")
		return errors.New("server return config file version not match baizContext version")
	}
	ctx.Config = config
	e = saveConfig()
	if nil != e {
		return e
	}
	if context.GetContext().IsIpfsStart {
		autoGather.SyncConfig()
		rummodedetect.SyncConfig()
		return SyncIpfsConfig()
	}

	return nil
}

//失败睡眠60s在试,重试3次
func getServerConfigWithRetry() (*config.Config, error) {
	var cfg *config.Config
	var err error
	for i := 0; i < 3; i++ {
		cfg, err = getServerConfig()
		if nil == err {
			return cfg, nil
		}
		time.Sleep(60 * time.Second)
	}
	return cfg, err
}

//从服务端获取配置
func getServerConfig() (*config.Config, error) {
	ctx := context.GetContext()
	now := time.Now()
	zoneName, offset := now.Zone()
	reqParam := map[string]string{
		"peer_id":         context.GetPeerID(),
		"goarch":          runtime.GOARCH,
		"goos":            runtime.GOOS,
		"version":         ctx.Version,
		"run_mode":        runmodestat.GetCurrentRunMode(),
		"startParams":     context.GetStartParamStr(),
		"dynamicParams":   context.GetDynamicParamStr(),
		"local_timezone":  zoneName + "|" + strconv.Itoa(offset),
		"local_timestamp": strconv.FormatInt(now.Unix(), 10),
		"local_timestr":   context.TransTimeStr(now),
	}

	reqData, _ := json.Marshal(reqParam)
	var errInfo string
	for i := 0; i < len(ctx.ConfigURLs); i++ {
		configURL := ctx.ConfigURLs[i]
		httpReq, err := http.NewRequest("GET", configURL, strings.NewReader(string(reqData)))
		if nil != err {
			return nil, err
		}
		httpRes, err := http.DefaultClient.Do(httpReq)
		//如果执行失败，执行下一个url
		if nil != err {
			errInfo = err.Error()
			fmt.Fprintf(os.Stderr, "error getConfig path:%s  err:%s\n", configURL, err)
			continue
		}
		defer httpRes.Body.Close()
		//如果状态码不为200，继续用下一个url
		status := httpRes.StatusCode
		if 200 != status {
			errInfo = "status:" + strconv.Itoa(status)
			fmt.Fprintf(os.Stderr, "error getConfig path:%s  status:%v\n", configURL, status)
			continue
		}
		resp, err := ioutil.ReadAll(httpRes.Body)
		//读取响应失败，继续用下一个url
		if nil != err {
			errInfo = err.Error()
			fmt.Fprintf(os.Stderr, "error getConfig path:%s  err:%s\n", configURL, err)
			continue
		}
		config := &config.Config{}
		err = json.Unmarshal(resp, config)
		//解码失败
		if nil != err {
			fmt.Fprintf(os.Stderr, "error getConfig path:%s  err:%s\n", configURL, err)
			errInfo = err.Error()
			continue
		}
		return config, nil
	}
	return nil, errors.New("getServerConfig cannot get config,err:" + errInfo)

}

//保存配置到本地
func saveConfig() error {
	ctx := context.GetContext()
	file, err := os.Create(ctx.BaizeConfigPath)
	if nil != err {
		fmt.Fprintf(os.Stderr, "create baize config file error: %s\n", err)
		return err
	}
	fileContent, e := json.Marshal(ctx.Config)
	if e != nil {
		fmt.Fprintf(os.Stderr, "error transform config file to josn: %s\n", err)
		return e
	}
	_, err = file.Write(fileContent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "save baize config file error: %s\n", err)
		return err
	}
	return nil
}

//SyncIpfsConfig 同步ipfs配置
func SyncIpfsConfig() error {
	ctx := context.GetContext()
	ipfsCfgPath := filepath.Join(ctx.RepoPath, "config")
	_, err := os.Stat(ipfsCfgPath)
	if nil != err {
		return err
	}
	currCfg, err := serialize.Load(ipfsCfgPath)
	if err != nil {
		return err
	}
	newCfg := ctx.Config.IpfsConfig
	currCfg.Datastore.StorageMax = newCfg.StorageMax
	currCfg.Datastore.StorageGCWatermark = newCfg.StorageGCWatermark
	currCfg.Datastore.GCPeriod = newCfg.GCPeriod
	currCfg.Bootstrap = newCfg.Bootstrap
	currCfg.Reprovider.Interval = newCfg.ReprovideInterval
	return serialize.WriteConfigFile(ipfsCfgPath, currCfg)
}

//Time2SyncConfig 定时10分钟从服务端拉取1次配置，更新本地
func Time2SyncConfig() {
	timer := time.NewTimer(getSyncConfigPeriod())
	for {
		select {
		case <-timer.C:
			err := SyncServerConfig()
			if nil != err {
				fmt.Fprintf(os.Stderr, "time to SyncServerConfig error: %s\n", err)
			}
			timer.Reset(getSyncConfigPeriod())
		case <-context.GetContext().BaizeCtx.Done():
			timer.Stop()
			fmt.Println("time2SyncConfig stop")
			return
		}
	}
}
func getSyncConfigPeriod() time.Duration {
	period, err := time.ParseDuration(context.GetContext().Config.CfgUpdatePeriod)
	if err != nil {
		period = 600 * time.Second
	}
	return period
}
