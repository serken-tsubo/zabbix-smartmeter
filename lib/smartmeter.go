package mpsm

import (
	"encoding/binary"
	"encoding/json" // 追加
	"flag"
	"fmt" // 追加
	"io"
	"log"
	"log/syslog"
	"os"
	"path"
	"time"

	smartmeter "github.com/hnw/go-smartmeter"
	mp "github.com/mackerelio/go-mackerel-plugin"
)

// ... (途中省略: SmartmeterPlugin構造体やメソッドはそのまま) ...

// Do the plugin
func Do() {
	var (
		optPrefix         = flag.String("metric-key-prefix", "smartmeter", "Metric key prefix")
		optTempfile       = flag.String("tempfile", "", "Temp file name")
		optBRouteID       = flag.String("id", "", "B-route ID")
		optBRoutePassword = flag.String("password", "", "B-route password")
		optSerialPort     = flag.String("device", "", "Path to serial port")
		optChannel        = flag.String("channel", "", "channel")
		optIPAddr         = flag.String("ipaddr", "", "IP address")
		optDualStackSK    = flag.Bool("dse", false, "Enable Dual Stack Edition (DSE) SK command")
		optScanMode       = flag.Bool("scan", false, "scan mode")
		optVerbosity      = flag.Int("verbosity", 1, "Verbosity (0:quiet, 3:debug)")
		optJson           = flag.Bool("json", false, "Output in JSON format for Zabbix") // 追加
	)

	flag.Parse()

	var writer io.Writer
	writer = os.Stdout
	if !*optScanMode {
		// JSONモードの時もログが標準出力に混ざらないようにsyslogを使用する
		var err error
		tag := path.Base(os.Args[0])
		writer, err = syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, tag)
		if err != nil {
			panic(err)
		}
	}

	// ... (スマートメーターのオープン処理はそのまま) ...
	dev, err := smartmeter.Open(
		*optSerialPort,
		smartmeter.ID(*optBRouteID),
		smartmeter.Password(*optBRoutePassword),
		smartmeter.Channel(*optChannel),
		smartmeter.IPAddr(*optIPAddr),
		smartmeter.DualStackSK(*optDualStackSK),
		smartmeter.Verbosity(*optVerbosity),
		smartmeter.Logger(log.New(writer, "", 0)),
	)
	if err != nil {
		panic(err)
	}

	p := SmartmeterPlugin{
		Prefix:   *optPrefix,
		dev:      dev,
		ScanMode: *optScanMode,
	}

	// 追加: JSONモードなら値を取得してJSON出力して終了
	if *optJson {
		metrics, err := p.FetchMetrics()
		if err != nil {
			// エラー時はログに出して終了（Zabbix側では取得不可となる）
			log.Printf("Failed to fetch metrics: %v", err)
			os.Exit(1)
		}
		
		// JSONに変換して標準出力へ
		jsonBytes, err := json.Marshal(metrics)
		if err != nil {
			log.Printf("Failed to marshal json: %v", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes))
		return
	}

	// Mackerelプラグインとしての動作
	plugin := mp.NewMackerelPlugin(p)
	plugin.Tempfile = *optTempfile
	plugin.Run()
}
