package ipfscfg

//IpfsConfig 针对ipfs的配置
type IpfsConfig struct {
	Bootstrap []string

	StorageMax         string // in B, kB, kiB, MB, ...
	StorageGCWatermark int64  // in percentage to multiply on StorageMax
	GCPeriod           string // in ns, us, ms, s, m, h
	ReprovideInterval  string

	ModeConfig map[string]RunModeConfig

	RunModeDetectConfig map[string]RunModeDetectConfig

	MobileUseThreshold    int64
	MobileUseNumThreshold int64
}
