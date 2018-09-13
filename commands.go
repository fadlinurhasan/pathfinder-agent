package main

import (
	"errors"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/BaritoLog/go-boilerplate/srvkit"
	"github.com/pathfinder-cm/pathfinder-agent/agent"
	"github.com/pathfinder-cm/pathfinder-agent/daemon"
	"github.com/pathfinder-cm/pathfinder-go-client/pfclient"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func CmdAgent(ctx *cli.Context) {
	if ctx.Bool("verbose") == true {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}

	log.WithFields(log.Fields{
		"verbose": ctx.Bool("verbose"),
	}).Warn("Agent starting...")
	runAgent()
	srvkit.GracefullShutdown(func() {
		log.Warn("Agent stopping...")
	})
}

func runAgent() {
	hostname, _ := os.Hostname()
	ipAddress := getLocalIP()
	daemon, err := daemon.NewLXD(hostname, LXDSocketPath)
	if err != nil {
		log.Error("Cannot connect to container daemon")
		return
	}
	httpClient := &http.Client{
		Timeout: time.Second * 60,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 60 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 60 * time.Second,
		},
	}
	pfclient := pfclient.NewPfclient(
		PfCluster,
		PfClusterPassword,
		httpClient,
		PfServerAddr,
		PfApiPath,
	)

	// Self Register
	ok, _ := pfclient.Register(hostname, ipAddress)
	if !ok {
		panic(errors.New("Cannot register to pathfinder server, please check your configuration."))
	}

	provisionAgent := agent.NewProvisionAgent(hostname, daemon, pfclient)
	go provisionAgent.Run()

	metricsAgent := agent.NewMetricsAgent(hostname, pfclient)
	go metricsAgent.Run()
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
