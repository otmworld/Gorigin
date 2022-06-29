package main

import (
	"flag"
	"log"
	"micro/network"
	"micro/network/watch"
	"os"
	"os/signal"
)

var (
	tcpPort  uint64
	udpPort  uint64
	httpPort uint64
	envPath  string
)

func main() {
	flag.Uint64Var(&tcpPort, "tcp", 8080, "tcp port")
	flag.Uint64Var(&udpPort, "udp", 8081, "tcp port")
	flag.Uint64Var(&httpPort, "http", 8082, "tcp port")
	flag.StringVar(&envPath, "env", "./", "function env file path")
	flag.Parse()

	config := &network.WatcherConfig{
		TcpPort:    tcpPort,
		UdpPort:    udpPort,
		HttpPort:   httpPort,
		ConfigPath: envPath,
	}
	if err := watch.NewWatcher(config); err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	singer := <-c
	log.Println("Signal: ", singer)
}
