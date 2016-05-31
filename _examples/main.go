package main

import (
	"github.com/bluven/etcd-election"
	//"github.com/koding/multiconfig"
	log "github.com/Sirupsen/logrus"
)


func main() {
	config := election.Config{
		Servers: []string{"127.0.0.1:2379"},
		Key: "/eonscheduer/scheduler/",
		Name: "test",
		Ttl: 30,
		Sleep: 5,
		Handler: func(isMaster bool) {
			if isMaster {
				log.Info("I'm the master")
			}
		},
	}

	election, err := election.New(config)

	if err != nil {
		log.Fatalln("Failed to create etcd election: ", err)
	}

	election.Start()
}

