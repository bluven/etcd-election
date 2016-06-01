package main

import (
	"time"

	etcd "github.com/coreos/etcd/client"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

func main() {

	log.SetLevel(log.DebugLevel)

	client, err := etcd.New(etcd.Config{Endpoints: []string{"http://127.0.0.1:2379"}})

	if err != nil {
		log.Fatalln(err)
	}

	keysApi := etcd.NewKeysAPI(client)

	resp, err := keysApi.Set(context.TODO(), "/test", "test", &etcd.SetOptions{
		//TTL: 10 * time.Second,
	})

	log.Debugf("%v", resp)
	log.Debugf("%v", resp.Node)

	resp, err = keysApi.Get(context.TODO(), "/test", nil)

	log.Debugf("%v", resp)
	log.Debugf("%#v", resp.Node)

	<-time.After(11 * time.Second)

	resp, err = keysApi.Get(context.TODO(), "/test", nil)

	log.Debugf("%v", resp)
	log.Debugf("%#v", resp.Node)

}
