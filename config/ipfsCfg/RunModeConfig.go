package ipfscfg

type RunModeConfig struct {
	DhtClient                bool
	BitSwapClient            bool
	EnableRelayHop           bool
	EnableDownLoad           bool
	DisableDownLoadRespCode  int
	EnableRelayAddr          bool
	MinConn                  int
	MaxConn                  int
	LimitRefreshRoutingTable bool
	LimitRoutTableNum        int
}
