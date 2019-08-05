package config

import (
	datagather "baize/config/datagatherCfg"
	download "baize/config/downloadCfg"
	ipfs "baize/config/ipfsCfg"
)

//Config 白泽模块配置对象
type Config struct {
	Version         string
	CfgUpdatePeriod string
	DataReport      datagather.DataReport
	IpfsConfig      ipfs.IpfsConfig
	DownloadConfig  download.DownloadConfig
}
