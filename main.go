package main

import (
	"flag"
	"fmt"
	"github.com/siddontang/redis-failover/failover"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var configFile = flag.String("config", "", "failover config file")
var addr = flag.String("addr", ":11000", "failover http listen addr")
var checkInterval = flag.Int("check_interval", 1000, "check master alive every n millisecond")
var maxDownTime = flag.Int("max_down_time", 3, "max down time for a master, after that, we will do failover")

var masters = flag.String("masters", "", "redis master need to be monitored, seperated by comma")
var mastersState = flag.String("masters_state", "", "new or existing for raft, if new, we will depracted old saved masters")

var broker = flag.String("broker", "", "broker for cluster, now is raft or zk")

var raftDataDir = flag.String("raft_data_dir", "./var/store", "raft data store path")
var raftLogDir = flag.String("raft_log_dir", "./var/log", "raft log store path")
var raftAddr = flag.String("raft_addr", "", "raft listen addr, if empty, we will disable raft")
var raftCluster = flag.String("raft_cluster", "", "raft cluster,vseperated by comma")
var raftClusterState = flag.String("raft_cluster_state", "", "new or existing, if new, we will deprecate old saved cluster and use new")

var zkAddr = flag.String("zk_addr", "", "zookeeper address, seperated by comma")
var zkPath = flag.String("zk_path", "/zk/redis/failover", "base directory in zk, prefix must be /zk")

func main() {
	flag.Parse()

	var c *failover.Config
	var err error
	if len(*configFile) > 0 {
		c, err = failover.NewConfigWithFile(*configFile)
		if err != nil {
			fmt.Printf("load failover config %s err %v", *configFile, err)
			return
		}
	} else {
		fmt.Printf("no config file, use default config")
		c = new(failover.Config)
		c.Addr = ":11000"
	}

	if *checkInterval > 0 {
		c.CheckInterval = *checkInterval
	}

	if *maxDownTime > 0 {
		c.MaxDownTime = *maxDownTime
	}

	if len(*raftAddr) > 0 {
		c.Raft.Addr = *raftAddr
	}

	if len(*raftDataDir) > 0 {
		c.Raft.DataDir = *raftDataDir
	}

	if len(*raftLogDir) > 0 {
		c.Raft.LogDir = *raftLogDir
	}

	seps := strings.Split(*raftCluster, ",")
	if len(seps) > 0 && len(seps[0]) > 0 {
		c.Raft.Cluster = seps
	}

	if len(*raftClusterState) > 0 {
		c.Raft.ClusterState = *raftClusterState
	}

	seps = strings.Split(*zkAddr, ",")
	if len(seps) > 0 && len(seps[0]) > 0 {
		c.Zk.Addr = seps
	}

	if len(*zkPath) > 0 {
		c.Zk.BaseDir = *zkPath
	}

	if len(*broker) > 0 {
		c.Broker = *broker
	} else if len(c.Raft.Addr) > 0 {
		// use raft
		c.Broker = "raft"
	} else if len(c.Zk.Addr) > 0 {
		c.Broker = "zk"
	}

	seps = strings.Split(*masters, ",")
	if len(seps) > 0 && len(seps[0]) > 0 {
		c.Masters = seps
	}

	if len(*mastersState) > 0 {
		c.MastersState = *mastersState
	}

	app, err := failover.NewApp(c)
	if err != nil {
		fmt.Printf("new failover app error %v", err)
		return
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		<-sc
		app.Close()
	}()

	app.Run()
}
