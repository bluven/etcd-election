package main

import (
	"time"

	"github.com/bluven/etcd-election"
	"github.com/koding/multiconfig"
	log "github.com/Sirupsen/logrus"
)


func main() {

	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.JSONFormatter{})

	config := election.Config{
		Servers: []string{"http://localhost:2379"},
		Key: "/eonscheduler/scheduler",
		Name: "panic",
		Ttl: 30 * time.Second,
		Sleep: 5 * time.Second,
		Handler: func(isMaster bool) {
			if isMaster {
				log.Info("I'm the master")
			}
		},
	}

	multiconfig.MustLoad(&config)

	election, err := election.New(config)

	if err != nil {
		log.Fatalln("Failed to create etcd election: ", err)
	}

	err = election.Test("/eonscheduler/test")

	if err != nil {
		log.Fatalln("Failed to connection to etcd servers: ", err)
	}

	election.Start()
}

