// Utility to perform master election/failover using etcd.

package election

import (
	"sync"
	"time"
	"errors"

	etcd "github.com/coreos/etcd/client"
	log "github.com/golang/glog"
	"golang.org/x/net/context"
)

type Handle func(isMaster bool)

type Election struct {
	running bool
	mu      *sync.RWMutex

	servers   []string
	key       string
	whoami    string
	ttl       time.Duration
	sleep     time.Duration
	lastLease time.Time

	handle Handle
}

type Config struct {
	Servers []string
	Key     string
	Name    string
	Ttl     time.Duration
	Sleep   time.Duration
	Handler Handle
}

func New(config Config) (*Election, error) {

	if len(config.Servers) == 0 {
		return nil, errors.New("Etcd servers cannot be empty or nil")
	}

	if config.Handler == nil {
		return nil, errors.New("The handler cannnot be nil")
	}

	// Todo: Need more validation
	if config.Key == "" {
		return nil, errors.New("Key cannot be blank or nil")
	}

	// Todo: Need more validation
	if config.Name == "" {
		return nil, errors.New("Name cannot be blank or nil")
	}

	return &Election{
		mu:      &sync.RWMutex{},
		servers: config.Servers,
		key:     config.Key,
		whoami:  config.Name,
		ttl:     config.Ttl,
		sleep:   config.Sleep,
		handle:  config.Handler,
	}, nil
}

func (self *Election) Start() {

	self.mu.Lock()

	if self.running {
		self.mu.Unlock()
		log.Errorln("The lock is already running.")
		return
	}

	self.running = true
	self.mu.Unlock()

	client, err := etcd.New(etcd.Config{Endpoints: self.servers})

	if err != nil {
		log.Fatalf("misconfigured etcd: %v", err)
	}

	for self.isRunning() {
		self.leaseAndUpdateLoop(client)
	}
}

func (self *Election) Stop() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.running = false
}

func (self *Election) isRunning() bool {
	self.mu.Lock()
	defer self.mu.Unlock()

	return self.running
}

// runs the election loop. never returns.
func (self *Election) leaseAndUpdateLoop(etcdClient etcd.Client) {
	for {
		master, err := self.acquireOrRenewLease(etcdClient)
		if err != nil {
			log.Errorf("Error in master election: %v", err)
			if time.Now().Sub(self.lastLease) < self.ttl {

				continue
			}
			// Our lease has expired due to our own accounting, pro-actively give it
			// up, even if we couldn't contact etcd.
			log.Infof("Too much time has elapsed, giving up lease.")

			master = false
		}

		self.handle(master)

		time.Sleep(self.sleep)
	}
}

// acquireOrRenewLease either races to acquire a new master lease, or update the existing master's lease
// returns true if we have the lease, and an error if one occurs.
// TODO: use the master election utility once it is merged in.
func (self *Election) acquireOrRenewLease(etcdClient etcd.Client) (bool, error) {

	keysAPI := etcd.NewKeysAPI(etcdClient)
	resp, err := keysAPI.Get(context.TODO(), self.key, nil)

	if err != nil {

		if etcd.IsKeyNotFound(err) {
			// there is no current master, try to become master, create will fail if the key already exists
			opts := etcd.SetOptions{
				TTL:       time.Duration(self.ttl) * time.Second,
				PrevExist: "",
			}
			_, err := keysAPI.Set(context.TODO(), self.key, self.whoami, &opts)
			if err != nil {
				return false, err
			}
			self.lastLease = time.Now()

			return true, nil
		}
		return false, err
	}

	if resp.Node.Value == self.whoami {
		log.Infof("key already exists, we are the master (%s)", resp.Node.Value)
		// we extend our lease @ 1/2 of the existing TTL, this ensures the master doesn't flap around
		if resp.Node.Expiration.Sub(time.Now()) < time.Duration(self.ttl/2)*time.Second {
			opts := etcd.SetOptions{
				TTL:       time.Duration(self.ttl) * time.Second,
				PrevValue: self.whoami,
				PrevIndex: resp.Node.ModifiedIndex,
			}
			_, err := keysAPI.Set(context.TODO(), self.key, self.whoami, &opts)
			if err != nil {
				return false, err
			}
		}
		self.lastLease = time.Now()
		return true, nil
	}

	log.Infof("key already exists, the master is %s, sleeping.", resp.Node.Value)
	return false, nil
}
