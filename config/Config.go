package config

import (
	datagather "github.com/lenovo-baize/baize/config/datagatherCfg"
	download "github.com/lenovo-baize/baize/config/downloadCfg"
	ipfs "github.com/lenovo-baize/baize/config/ipfsCfg"
)

//Config 白泽模块配置对象
type Config struct {
	Version         string
	CfgUpdatePeriod string
	DataReport      datagather.DataReport
	IpfsConfig      ipfs.IpfsConfig
	DownloadConfig  download.DownloadConfig
}
