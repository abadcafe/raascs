package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/abadcafe/autosplitfile"
	"github.com/abadcafe/raascs/resp"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"
)

const serverLogFilename = "server.log"

var (
	versionStr = "unknown"
	configFile = flag.String("f", "conf/server.json", "configFile")
	logDir     = flag.String("l", "logs", "logDir")
)

var config = struct {
	MaxLogFileSize int64
	MaxLogDuration string
	ListenAddr     string
}{
	MaxLogFileSize: 1048576 * 1024,
	MaxLogDuration: "24h",
	ListenAddr:     ":6380",
}

func readConfig() error {
	cf, err := os.Open(*configFile)
	if err != nil {
		return err
	}

	err = json.NewDecoder(cf).Decode(&config)
	if err != nil {
		return err
	}

	return nil
}

func registerStopFunc(stopFunc func()) chan struct{} {
	waitStopSignal := make(chan os.Signal, 1)
	signal.Notify(waitStopSignal, syscall.SIGINT, syscall.SIGTERM)

	waitStopFunc := make(chan struct{})
	go func() {
		occurredSignal := <-waitStopSignal
		log.WithField("signal", occurredSignal.String()).Info("stop signal received")

		stopFunc()
		close(waitStopFunc)
	}()

	return waitStopFunc
}

func main() {
	showVersion := flag.Bool("version", false, "version")
	flag.Parse()

	if *showVersion {
		fmt.Println(versionStr)
		return
	}

	log.Infof("config file: %s, logDir: %s", *configFile, *logDir)
	err := readConfig()
	if err != nil {
		if !os.IsNotExist(err) {
			log.WithError(err).Fatal("read config failed")
		}

		log.WithError(err).Warn("config file not exist, use default config")
	}

	logFile, err := autosplitfile.New(&autosplitfile.FileOptions{
		PathPrefix: path.Join(*logDir, serverLogFilename),
		MaxSize:    config.MaxLogFileSize,
		MaxTime:    config.MaxLogDuration,
	})
	if err != nil {
		log.WithError(err).Fatal("create server log file failed")
	}

	log.RegisterExitHandler(func() { _ = logFile.Close() })
	log.SetOutput(logFile)
	log.SetFormatter(&log.TextFormatter{})
	log.WithField("config", fmt.Sprintf("%+v", config)).Info("$$$$$$$ starting server ... $$$$$$$")

	listener, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		log.WithError(err).WithField("addr", config.ListenAddr).Fatal("listen failed")
	}

	server := resp.NewServer(listener)
	waitGracefulStop := registerStopFunc(server.GracefulStop)

	log.Info("serving...")
	err = server.Serve()
	if err != nil {
		log.WithError(err).Fatal("server serve failed")
	}

	<-waitGracefulStop
	log.Info("server stopped")
	_ = logFile.Close()
}
