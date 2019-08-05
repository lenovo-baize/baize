package autoGather

import (
	"fmt"
	humanize "github.com/dustin/go-humanize"
	"github.com/prometheus/client_golang/prometheus"
	swarm "github.com/libp2p/go-libp2p-swarm"
	"net"
	"os"
	"strconv"
	"strings"

	context "github.com/lenovo-baize/baize/context"
)

//BuildIpfsMetricsData 采集ipfs的指标数据
func BuildIpfsMetricsData() map[string]string {
	data := make(map[string]string)
	if !context.GetContext().IsIpfsStart {
		return data
	}
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		fmt.Fprintf(os.Stderr, "An error has occurred during metrics collection: %s\n", err)
		data["get_metric_error"] = err.Error()
		return nil
	}
	for _, mf := range mfs {
		name := *mf.Name
		for i := 0; i < len(mf.Metric); i++ {
			switch mf.GetType().String() {
			case "COUNTER":
				data[name] = strconv.FormatFloat(*mf.Metric[i].Counter.Value, 'f', -1, 64)
				break
			case "GAUGE":
				data[name] = strconv.FormatFloat(*mf.Metric[i].Gauge.Value, 'f', -1, 64)
				break
			case "SUMMARY":
				if len(mf.Metric) == 1 {
					data[name+"_sample_count"] = strconv.FormatUint(*mf.Metric[i].Summary.SampleCount, 10)
					data[name+"_sample_sum"] = strconv.FormatFloat(*mf.Metric[i].Summary.SampleSum, 'f', -1, 64)
					for j := 0; j < len(mf.Metric[i].Summary.Quantile); j++ {
						quantile := mf.Metric[i].Summary.Quantile[j]
						key := name + "_quantile_" + strconv.FormatFloat(*quantile.Quantile, 'f', -1, 64)
						data[key] = strconv.FormatFloat(*quantile.Value, 'f', -1, 64)
					}
					break
				}

				for j := 0; j < len(mf.Metric); j++ {
					metric := mf.Metric[j]
					data[name+"_"+*metric.Label[0].Value+"_sample_count"] = strconv.FormatUint(*metric.Summary.SampleCount, 10)
					data[name+"_"+*metric.Label[0].Value+"_sample_sum"] = strconv.FormatFloat(*metric.Summary.SampleSum, 'f', -1, 64)
					for j := 0; j < len(mf.Metric[i].Summary.Quantile); j++ {
						quantile := mf.Metric[i].Summary.Quantile[j]
						key := name + "_" + *metric.Label[0].Value + "_quantile_" + strconv.FormatFloat(*quantile.Quantile, 'f', -1, 64)
						data[key] = strconv.FormatFloat(*quantile.Value, 'f', -1, 64)
					}
				}
				break
			case "UNTYPED":
				data[name] = strconv.FormatFloat(*mf.Metric[i].Untyped.Value, 'f', -1, 64)
				break
			case "HISTOGRAM":
				data[name+"_sample_count"] = strconv.FormatUint(*mf.Metric[i].Histogram.SampleCount, 10)
				data[name+"_sample_sum"] = strconv.FormatFloat(*mf.Metric[i].Histogram.SampleSum, 'f', -1, 64)
				for j := 0; j < len(mf.Metric[i].Histogram.Bucket); j++ {
					bucket := mf.Metric[i].Histogram.Bucket[j]
					key := name + "_bucket_" + strconv.FormatFloat(*bucket.UpperBound, 'f', -1, 64)
					data[key] = strconv.FormatUint(*bucket.CumulativeCount, 10)
				}
				break
			default:
				break
			}
		}

	}
	storageUsage, err := context.GetContext().IpfsNode.Repo.GetStorageUsage()
	if nil != err {
		fmt.Fprintf(os.Stderr, "An error has occurred during storage_usage collection: %s\n", err)
		data["get_storage_usage_error"] = err.Error()
	}
	data["storage_usage"] = strconv.FormatUint(storageUsage, 10)

	bs := context.GetContext().IpfsNode.Reporter.GetBandwidthTotals()
	bandWidthStr := "total:" + humanize.Bytes(uint64(bs.TotalIn)) + "," + humanize.Bytes(uint64(bs.TotalOut)) + "," + humanize.Bytes(uint64(bs.RateIn)) + "," + humanize.Bytes(uint64(bs.RateOut))
	//pbs := context.GetContext().IpfsNode.Reporter.GetBandwidthForAllProtocol()
	//for k, v := range pbs {
	//	bandWidthStr = bandWidthStr + "|" + k + ":" + v["total_in"] + "," + v["total_out"] + "," + v["rate_in"] + "," + v["rate_out"]
	//}
	data["band_width"] = bandWidthStr
	return data
}
func mac() string {
	// 获取本机的MAC地址
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, inter := range interfaces {
		mac := inter.HardwareAddr //获取本机MAC地址
		return mac.String()
	}
	return ""
}

//BuildConnNumsData 构建连接数数据
func BuildConnNumsData() map[string]string {
	data := make(map[string]string)
	if !context.GetContext().IsIpfsStart {
		return data
	}
	conns := context.GetContext().IpfsNode.PeerHost.Network().Conns()
	data["total_conn"] = strconv.Itoa(len(conns))
	tcpNum := 0
	relayNum := 0
	otherNum := 0
	for i := range conns {
		conn := conns[i]
		conn1 := conn.(*swarm.Conn)
		connStr := conn1.String()
		if strings.Contains(connStr, "swarm.Conn[TCP]") {
			tcpNum++
			continue
		}
		if strings.Contains(connStr, "chan *relay.Conn=") {
			relayNum++
			continue
		}
		otherNum++
	}

	data["tcp_con_num"] = strconv.Itoa(tcpNum)
	data["relay_con_num"] = strconv.Itoa(relayNum)
	data["other_con_num"] = strconv.Itoa(otherNum)
	return data
}
