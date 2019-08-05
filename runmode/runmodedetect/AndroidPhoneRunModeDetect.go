package rummodedetect

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lenovo-baize/baize/context"
	"github.com/lenovo-baize/baize/runmode/runmodestat"
)

//上一次的移动流量
var LastMobileTraffic int64 = 0
var LastWIFITraffic int64 = 0
var preNetwork string

func DetectRunModeOnAndroidPhone() {
	detectRunMod()
	go time2CheckMobieTraffic()
	listenNetwork()
}

var runModeLock sync.Mutex

func time2CheckMobieTraffic() {
	timer := time.NewTimer(GetMobileTrafficCheckPeriod())
	for {
		select {
		case <-timer.C:
			//WIFI share模式才执行检测
			mobileTrafficCheck()
			timer.Reset(GetMobileTrafficCheckPeriod())
		case <-baizectx.GetContext().BaizeCtx.Done():
			timer.Stop()
			fmt.Println("time2SyncConfig stop")
			return
		}
	}
}

//在wifi share模式时，有2次发现使用了移动流量，则切换的client
var mobileTrafficNum int64 = 0

func mobileTrafficCheck() {
	runModeLock.Lock()
	defer runModeLock.Unlock()
	network := baizectx.GetNetwork()
	//非wifi 但是运行的是share模式，切换到client模式
	if "WIFI" != network && runmodestat.RUN_MODE_SHARE == runmodestat.GetCurrentRunMode() {
		fmt.Println(os.Stdout, time.Now().String()+"  time to mobileTrafficCheck curr network is "+network+" change to client")
		changeToClientMode("time to mobileTrafficCheck curr network is " + network + " change to client")
		mobileTrafficNum = 0
		return
	}
	if "WIFI" == strings.ToUpper(network) && runmodestat.RUN_MODE_SHARE == runmodestat.GetCurrentRunMode() {
		mobileTraffic, err := readMobileTraffic()
		if nil != err {
			fmt.Println(os.Stdout, time.Now().String()+"  time to mobileTrafficCheck err:"+err.Error())
			changeToClientMode("time call to read mobile traffic err:" + err.Error())
			return
		}
		useMobileTraffic := mobileTraffic - LastMobileTraffic
		tmpLastMobileTraffic := LastMobileTraffic
		LastMobileTraffic = mobileTraffic
		if useMobileTraffic > baizectx.GetContext().Config.IpfsConfig.MobileUseThreshold {
			mobileTrafficNum = mobileTrafficNum + 1
			if mobileTrafficNum > baizectx.GetContext().Config.IpfsConfig.MobileUseNumThreshold {
				fmt.Println(os.Stdout, time.Now().String()+"  time to mobileTrafficCheck mobile traffic limit LastMobileTraffic:"+strconv.FormatInt(tmpLastMobileTraffic, 10)+",currMobileTraffic:"+strconv.FormatInt(mobileTraffic, 10)+",useMobileTraffic:"+strconv.FormatInt(useMobileTraffic, 10)+",mobileTrafficNum:"+strconv.FormatInt(mobileTrafficNum, 10)+"  change client")
				changeToClientMode("mobile traffic limit LastMobileTraffic:" + strconv.FormatInt(tmpLastMobileTraffic, 10) + ",currMobileTraffic:" + strconv.FormatInt(mobileTraffic, 10) + ",useMobileTraffic:" + strconv.FormatInt(useMobileTraffic, 10) + ",mobileTrafficNum:" + strconv.FormatInt(mobileTrafficNum, 10) + "  change client")
				mobileTrafficNum = 0
				return
			}
			fmt.Println(os.Stdout, time.Now().String()+"  time to mobileTrafficCheck mobile traffic limit LastMobileTraffic:"+strconv.FormatInt(tmpLastMobileTraffic, 10)+",currMobileTraffic:"+strconv.FormatInt(mobileTraffic, 10)+",useMobileTraffic:"+strconv.FormatInt(useMobileTraffic, 10)+",mobileTrafficNum:"+strconv.FormatInt(mobileTrafficNum, 10)+" not change client")
			return
		}
		//fmt.Println(os.Stdout, time.Now().String()+"  time to mobileTrafficCheck normal,mobileTrafficNum:"+strconv.FormatInt(mobileTrafficNum, 10))
	} else {
		mobileTrafficNum = 0
	}
}

func detectRunMod() {
	network := baizectx.GetNetwork()
	if "WIFI" == strings.ToUpper(network) {
		runWifiMode()
		return
	}
	changeToClientMode("curr network is not wifi")
}
func runWifiMode() {
	mobileTraffic, err := readMobileTraffic()
	if nil != err {
		changeToClientMode("red mobileTraffic err:" + err.Error())
		return
	}
	LastMobileTraffic = mobileTraffic
	changeToShareMode("network is wifi LastMobileTraffic:" + strconv.FormatInt(LastMobileTraffic, 10))
}

func readMobileTraffic() (int64, error) {
	mobileInTraffic, mobileOutTraffic, _, _, err := getTraffic()
	if nil != err {
		return 0, err
	}
	currentMobileTraffic := mobileInTraffic + mobileOutTraffic
	if currentMobileTraffic < 0 {
		return 0, errors.New("MobileTraffic < 0")
	}
	return currentMobileTraffic, nil
}

//监听网络变化

func listenNetwork() {
	baizectx.ListenEvent(baizectx.DYNAMIC_PARAM_CHANGE, func() {
		runModeLock.Lock()
		defer runModeLock.Unlock()
		network := baizectx.GetNetwork()
		fmt.Println(os.Stdout, time.Now().String()+"listenNetwork network change,preNetwork:"+preNetwork+",currNetwork:"+network)
		//跟前1次网络没变化不处理，发现切换1次，会出现调用多次的情况
		if preNetwork == network {
			return
		}
		preNetwork = network
		//time.Sleep(3 * time.Second)
		detectRunMod()
	})
}
func GetMobileTrafficCheckPeriod() time.Duration {
	period, err := time.ParseDuration(baizectx.GetContext().Config.IpfsConfig.RunModeDetectConfig[baizectx.GetContext().StartParams["source"]].DetectPeriod)
	if err != nil {
		period = 30 * time.Second
	}
	return period
}

/*
* idx 			: 序号
* iface 		：代表流量类型（rmnet表示2G/3G, wlan表示Wifi流量,lo表示本地流量）
* acct_tag_hex 		：线程标记（用于区分单个应用内不同模块/线程的流量）
* uid_tag_int 		：应用uid,据此判断是否是某应用统计的流量数据
* cnt_set 		：应用前后标志位：1：前台， 0：后台
* rx_btyes 		：receive bytes 接受到的字节数
* rx_packets 		: 接收到的任务包数
* tx_bytes 		：transmit bytes 发送的总字节数
* tx_packets 		：发送的总包数
* rx_tcp_types 		：接收到的tcp字节数
* rx_tcp_packets 	：接收到的tcp包数
* rx_udp_bytes 		：接收到的udp字节数
* rx_udp_packets 	：接收到的udp包数
* rx_other_bytes 	：接收到的其他类型字节数
* rx_other_packets 	：接收到的其他类型包数
* tx_tcp_bytes 		：发送的tcp字节数
* tx_tcp_packets 	：发送的tcp包数
* tx_udp_bytes 		：发送的udp字节数
* tx_udp_packets 	：发送的udp包数
* tx_other_bytes 	：发送的其他类型字节数
* tx_other_packets 	：发送的其他类型包数
 */
func getTraffic() (inMobile int64, outMobile int64, inWifi int64, outWifi int64, errrs error) {
	defer func() {
		if r := recover(); r != nil {
			errrs = errors.New("Unknow panic")
			switch x := r.(type) {
			case string:
				errrs = errors.New(x)
			case error:
				errrs = x
			default:
			}
			fmt.Fprintf(os.Stderr, "error: %s\n", errrs)
		}
	}()
	uid := baizectx.GetContext().StartParams["uid"]
	//没有uid，使用client模式
	if "" == uid {
		return 0, 0, 0, 0, errors.New("android uid is empty")
	}
	f, err := os.Open("/proc/net/xt_qtaguid/stats")
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer f.Close()
	var totalInMobile int64 = 0
	var totalOutMobile int64 = 0
	var totalInWifi int64 = 0
	var totalOutWifi int64 = 0

	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		if strings.Contains(line, uid) {
			strs := strings.Split(line, " ")
			datain, err := strconv.ParseInt(strs[5], 10, 64)
			if err != nil {
				return 0, 0, 0, 0, err
			}
			dataout, err := strconv.ParseInt(strs[7], 10, 64)
			if err != nil {
				return 0, 0, 0, 0, err
			}
			if strings.Contains(strs[1], "rmnet") {
				totalInMobile += datain
				totalOutMobile += dataout
			} else if strings.Contains(strs[1], "wlan") {
				totalInWifi += datain
				totalOutWifi += dataout
			}
		}
		if err == nil {
			continue
		}

		if err != io.EOF {
			return 0, 0, 0, 0, err
		}
		return totalInMobile, totalOutMobile, totalInWifi, totalOutWifi, nil
	}
}

func GetAndroidPhoneTrafficData() map[string]string {
	dataMap := make(map[string]string)
	if "android-phone" == baizectx.GetContext().StartParams["source"] {
		mobileInTraffic, mobileOutTraffic, wifiInTraffic, wifiOutTraffic, err := getTraffic()
		dataMap["mobile_in"] = strconv.FormatInt(mobileInTraffic, 10)
		dataMap["mobile_out"] = strconv.FormatInt(mobileOutTraffic, 10)
		dataMap["wifi_in"] = strconv.FormatInt(wifiInTraffic, 10)
		dataMap["wifi_out"] = strconv.FormatInt(wifiOutTraffic, 10)
		dataMap["LastMobileTraffic"] = strconv.FormatInt(LastMobileTraffic, 10)
		dataMap["LastWIFITraffic"] = strconv.FormatInt(LastWIFITraffic, 10)
		if nil != err {
			dataMap["trafficerr"] = err.Error()
		}
		return dataMap
	}
	return dataMap
}
