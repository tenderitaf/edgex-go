//
// Copyright (c) 2017
// Cavium
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"flag"
	"fmt"
	"github.com/edgexfoundry/edgex-go/pkg/config"
	"os"
	"os/signal"
	"syscall"

	"github.com/edgexfoundry/edgex-go"
	"github.com/edgexfoundry/edgex-go/core/domain/models"
	"github.com/edgexfoundry/edgex-go/export/distro"

	"go.uber.org/zap"
)

var logger *zap.Logger

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting edgex export client", zap.String("version", edgex.Version))

	var (
		useConsul  = flag.String("consul", "", "Should the service use consul?")
		useProfile = flag.String("profile", "default", "Specify a profile other than default.")
	)
	flag.Parse()

	configuration := &distro.ConfigurationStruct{}
	err := config.LoadFromFile(*useProfile, configuration)
	if err != nil {
		logger.Error(err.Error(), zap.String("version", edgex.Version))
		return
	}

	//Determine if configuration should be overridden from Consul
	var consulMsg string
	if *useConsul == "y" {
		consulMsg = "Loading configuration from Consul..."
		err := distro.ConnectToConsul(*configuration)
		if err != nil {
			logger.Error(err.Error(), zap.String("version", edgex.Version))
			return //end program since user explicitly told us to use Consul.
		}
	} else {
		consulMsg = "Bypassing Consul configuration..."
	}

	logger.Info(consulMsg, zap.String("version", edgex.Version))

	err = distro.Init(*configuration, logger)

	logger.Info("Starting distro")
	errs := make(chan error, 2)
	eventCh := make(chan *models.Event, 10)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	// There can be another receivers that can be initialiced here
	distro.ZeroMQReceiver(eventCh)

	distro.Loop(errs, eventCh)

	logger.Info("terminated")
}
