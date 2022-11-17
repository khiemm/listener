package main

import (
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/khiemm/listener/devices/teltonika"
	"github.com/khiemm/listener/util"
	"github.com/spf13/viper"
	"github.com/thejerf/suture"
)

func init() {
	err := util.InitializeViper()
	if err != nil {
		panic(err)
	}
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func main() {
	connSupervisor := suture.New("connections", suture.Spec{
		FailureDecay:     float64(math.MaxInt64),
		FailureThreshold: float64(0.01),
		FailureBackoff:   time.Duration(math.MaxInt64),
		Log:              func(msg string) { log.Infof("suture: %s", msg) },
	})

	teltonikaServer := teltonika.MakeServer(viper.GetString("teltonika.address"), connSupervisor)
	
	supervisor := suture.NewSimple("root")
	supervisor.Add(connSupervisor)
	supervisor.Add(teltonikaServer)

	supervisor.ServeBackground()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	<-sigchan
	log.Info("Terminating")
	supervisor.Stop()
	log.Info("Terminated")
}