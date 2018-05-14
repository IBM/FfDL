/*
 * Copyright 2017-2018 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package coord

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/IBM/FfDL/commons/logger"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/clientv3util"
	"github.com/coreos/etcd/clientv3/namespace"
	etcdRecipes "github.com/coreos/etcd/contrib/recipes"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
)

const (
	timeout     = 10 * time.Second
)

//coordinator ... Thin wrapper to prevent exposing any of the etcd specific logic
type coordinator struct {
	cli    *clientv3.Client
	config *Config
}

//queueHandler is a wrapper to capture the state of a named queue
type queueHandler struct {
	queueName   string
	queue       *etcdRecipes.Queue
}

//valueSequenceHandler is a wrapper to capture the state of a value sequence
type valueSequenceHandler struct {
	coordinator *coordinator
	keyPrefix   string
}

//Coordinator ... interfacing declaring methods that can be consumed
type Coordinator interface {
	Close(log *logger.LocLoggingEntry)
	Get(path string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) ([]EtcdKVGetResponse, error)
	Put(path string, value string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) (EtcdKVPutResponse, error)
	PutIfKeyExists(path string, value string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) (bool, error)
	PutIfKeyMissing(path string, value string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) (bool, error)
	CompareAndSwap(path string, newValue string, expectedOldValue string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) (bool, error)
	DeleteKeyIfExists(path string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) (bool, error)
	DeleteKeyWithOpts(path string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) error
	WatchPath(ctx context.Context, path string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) clientv3.WatchChan
	GrantExpiringLease(leaseTimeout int64, log *logger.LocLoggingEntry) (*clientv3.LeaseGrantResponse, error)
	RefreshLease(leaseID clientv3.LeaseID, log *logger.LocLoggingEntry) (*clientv3.LeaseKeepAliveResponse, error)
	RevokeLease(leaseID clientv3.LeaseID, log *logger.LocLoggingEntry) error
	GetLeaseDetails(leaseID clientv3.LeaseID, log *logger.LocLoggingEntry) (*clientv3.LeaseTimeToLiveResponse, error)
	NewQueue(queueName string, log *logger.LocLoggingEntry) *queueHandler
	NewValueSequence(sequenceName string, log *logger.LocLoggingEntry) *valueSequenceHandler
}

// QueueHandler is a simple interface for a queue
type QueueHandler interface {
	Enqueue(message string, log *logger.LocLoggingEntry) error
	Dequeue(log *logger.LocLoggingEntry) (string, error)
}

// ValueSequence is an interface for a sequence of values, stored in temporal order
type ValueSequence interface {
	AddNew(value string, log *logger.LocLoggingEntry) error
	GetAll(log *logger.LocLoggingEntry) ([]string, error)
}

//Config ..config passed to the coordinator
type Config struct {
	Endpoints []string
	Prefix    string
	Cert      string
	Username  string
	Password  string
}

//EtcdKVGetResponse ...
type EtcdKVGetResponse struct {
	Key   string
	Value string
}

//EtcdKVPutResponse ...
type EtcdKVPutResponse struct {
	Key      string
	Value    string
	Revision int64
}

//NewCoordinator ... create a new instance of coordinator. Passes the error back to client in case error is encountered
func NewCoordinator(config Config, log *logger.LocLoggingEntry) (Coordinator, error) {

	log.Debugf("New coordinator request with endpoints %v , prefix %s, username %s, cert %s", config.Endpoints, config.Prefix, config.Username, config.Cert)
	etcdClient, err := connect(config, log)
	if err != nil {
		log.WithError(err).Errorf("Failed creating coordinator request with endpoints %v , prefix %s, username %s, cert %s ", config.Endpoints, config.Prefix, config.Username, config.Cert)
		return nil, err
	}

	coordinator := coordinator{
		cli:    etcdClient,
		config: &config,
	}
	return &coordinator, err
}

//Close ... close the underlying client
func (instance *coordinator) Close(log *logger.LocLoggingEntry) {
	log.Debugf("Closing coordinator")
	instance.cli.Close()
}

//Get ... get the value corresponding to the key. If an error is encountered then the error is propogated back. Use clientv3.WithLastRev() as options if only last revision is required
func (instance *coordinator) Get(path string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) ([]EtcdKVGetResponse, error) {

	res, nrerr := retry(2, 5*time.Second, "ETCD_GET", log, func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		result, err := instance.cli.Get(ctx, path, opts...)
		return result, err
	}, func(err error) bool {
		return handleError(err, log)
	})

	response, ok := res.(*clientv3.GetResponse)
	if nrerr != nil || !ok {
		log.WithError(nrerr).Errorf("Failed to get values for path %s with options %v ", path, opts)
		return nil, nrerr
	}

	log.Debugf("GET key with value %v and length %d", response.Kvs, len(response.Kvs))

	var result []EtcdKVGetResponse
	for _, val := range response.Kvs {
		result = append(result, EtcdKVGetResponse{
			Key:   string(val.Key),
			Value: string(val.Value),
		})
	}

	return result, nrerr

}

//Put ...put a given value against a key and return the last value of the key
func (instance *coordinator) Put(path string, value string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) (EtcdKVPutResponse, error) {

	res, nrerr := retry(2, 5*time.Second, "ETCD_PUT", log, func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		result, err := instance.cli.Put(ctx, path, value, opts...)
		return result, err
	}, func(err error) bool {
		return handleError(err, log)
	})

	response, ok := res.(*clientv3.PutResponse)
	if nrerr != nil || !ok {
		log.WithError(nrerr).Errorf("Failed to put values for path %s, value %s with options %v", path, value, opts)
		return EtcdKVPutResponse{}, nrerr
	}

	result := EtcdKVPutResponse{
		Revision: response.Header.Revision,
		Key:      path,
	}
	if response.PrevKv != nil {
		log.Debugf("Got previous value : %s for the key %s", response.PrevKv.Value, response.PrevKv.Key)
		result.Value = string(response.PrevKv.Value)
	}
	return result, nrerr
}

//PutIfKeyExists ...put value if the key already exists
func (instance *coordinator) PutIfKeyExists(path string, value string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	tx := instance.cli.Txn(ctx)
	resp, err := tx.
		If(clientv3util.KeyExists(path)).
		Then(clientv3.OpPut(path, value, opts...)).
		Commit()
	if err != nil {
		log.WithError(err).Errorf("Exception while performing a transaction to update key %s , with value %s and options %v.", path, value, opts)
		return false, err
	}
	log.Debugf("Completed the PutIfKeyExists operation for key %s , value %s and opts %v with result %t", path, value, opts, resp.Succeeded)
	return resp.Succeeded, err
}

//PutIfKeyMissing ...put a value only if key is missing
func (instance *coordinator) PutIfKeyMissing(path string, value string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	tx := instance.cli.Txn(ctx)
	resp, err := tx.
		If(clientv3util.KeyMissing(path)).
		Then(clientv3.OpPut(path, value, opts...)).
		Commit()
	if err != nil {
		log.WithError(err).Errorf("Exception while performing a transaction to update key %s , with value %s and options %v.", path, value, opts)
		return false, err
	}
	//log.Debugf("Completed the  PutIfKeyMissing operation for key %s , value %s and opts %v with result %t", path, value, opts, resp.Succeeded)
	return resp.Succeeded, err
}

//CompareAndSwap ...replaces an existing value with a new value
func (instance *coordinator) CompareAndSwap(path string, newValue string, expectedOldValue string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	tx := instance.cli.Txn(ctx)

	resp, err := tx.
		If(clientv3.Compare(clientv3.Value(path), "=", expectedOldValue)).
		Then(clientv3.OpPut(path, newValue, opts...)).
		Commit()

	if err != nil {
		log.WithError(err).Errorf("Exception while performing a transaction to compare and swap key %s , with newvalue %s , existingvalue %s and options %v", path, newValue, expectedOldValue, opts)
		return false, err
	}

	log.Debugf("Completed the  CompareAndSwap operation for key %s , newvalue %s , existingvalue %s and opts %v with result %t", path, newValue, expectedOldValue, opts, resp.Succeeded)
	return resp.Succeeded, err
}

//DeleteKeyIfExists ..delete key if it exists. Can do recursive delete, or pattern matching delete with prefix. use DeleteKeyWithOpts for that
func (instance *coordinator) DeleteKeyIfExists(path string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tx := instance.cli.Txn(ctx)

	resp, err := tx.
		If(clientv3util.KeyExists(path)).
		Then(clientv3.OpDelete(path, opts...)).
		Commit()

	if err != nil {
		log.WithError(err).Errorf("Exception while performing a transaction to delete key %s and options %v.", path, opts)
		return false, err
	}
	log.Debugf("Completed the  DeleteKey operation for key %s , opts %v with result %t", path, opts, resp.Succeeded)
	return resp.Succeeded, err
}

//DeleteKeyWithOpts .. with options provided. Does not do a ifExists check. Use the passed options to delete recursively
func (instance *coordinator) DeleteKeyWithOpts(path string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) error {

	_, nrerr := retry(2, 5*time.Second, "ETCD_GET", log, func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		result, err := instance.cli.Delete(ctx, path, opts...)
		return result, err
	}, func(err error) bool {
		return handleError(err, log)
	})

	if nrerr != nil {
		log.WithError(nrerr).Errorf("Exception while performing delete key %s and options %v.", path, opts)
		return nrerr
	}
	log.Debugf("Completed the  DeleteKey operation for key %s , opts %v ", path, opts)
	return nrerr
}

//WatchPath ...watches a given path. Optinally can recursively watch with prefix
func (instance *coordinator) WatchPath(ctx context.Context, path string, log *logger.LocLoggingEntry, opts ...clientv3.OpOption) clientv3.WatchChan {

	log.Debugf("setting up watch against path %s with options %v", path, opts)
	rch := instance.cli.Watch(ctx, path, opts...)
	return rch
}

// GrantExpiringLease ..creates an ephemeral node and just prods it once. caller needs to explicitly keep the lease alive
func (instance *coordinator) GrantExpiringLease(leaseTimeout int64, log *logger.LocLoggingEntry) (*clientv3.LeaseGrantResponse, error) {
	lease, err := instance.cli.Grant(context.TODO(), leaseTimeout)
	if err != nil {
		log.WithError(err).Errorf("Failed to grant the lease")
		return nil, err
	}

	log.Debugf("setting up lease %d with timeout %d", lease.ID, leaseTimeout)
	return lease, err
}

// RefreshLease ..prod lease once to keep it alive
func (instance *coordinator) RefreshLease(leaseID clientv3.LeaseID, log *logger.LocLoggingEntry) (*clientv3.LeaseKeepAliveResponse, error) {
	//prod once to keep alive
	leaseResponse, kaerr := instance.cli.KeepAliveOnce(context.TODO(), leaseID)
	log.Debugf("Refreshed lease for id %d and got the refreshed lease with id %d and TTL: %d ", leaseID, leaseResponse.ID, leaseResponse.TTL)
	return leaseResponse, kaerr
}

// GetLeaseDetails ...details of lease, including TTL and keys associated
func (instance *coordinator) GetLeaseDetails(leaseID clientv3.LeaseID, log *logger.LocLoggingEntry) (*clientv3.LeaseTimeToLiveResponse, error) {

	res, nrerr := retry(2, 5*time.Second, "ETCD_LEASE_DETAILS", log, func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		result, err := instance.cli.TimeToLive(ctx, leaseID, clientv3.WithAttachedKeys())
		return result, err
	}, func(err error) bool {
		return handleError(err, log)
	})

	leaseDetails, ok := res.(*clientv3.LeaseTimeToLiveResponse)
	if nrerr != nil || !ok {
		return nil, nrerr
	}
	log.Debugf("Got lease details %d : as %v and other details TTL: %d , Granted TTL : %d, %v", leaseID, leaseDetails, leaseDetails.TTL, leaseDetails.GrantedTTL, leaseDetails.Keys)
	return leaseDetails, nrerr
}

// RevokeLease ...revoke lease and deletes all the associated keys
func (instance *coordinator) RevokeLease(leaseID clientv3.LeaseID, log *logger.LocLoggingEntry) error {

	_, nrerr := retry(2, 5*time.Second, "ETCD_REVOKE_LEASE", log, func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		res, err := instance.cli.Revoke(ctx, leaseID)
		return res, err
	}, func(err error) bool {
		return handleError(err, log)
	})

	if nrerr != nil {
		log.WithError(nrerr).Errorf("Failed to revoke lease %d", leaseID)
		return nrerr
	}

	log.Debugf("successfully revoked lease %d", leaseID)
	return nrerr
}

// NewQueue ... creates a new queue with the given name
func (instance *coordinator) NewQueue(queueName string, log *logger.LocLoggingEntry) *queueHandler {
	log.Infof("Creating new message queue '%s'", queueName)
	queue := &queueHandler{
		queueName: queueName,
		queue:     etcdRecipes.NewQueue(instance.cli, queueName),
	}
	return queue
}

// Enqueue ... puts a message to the queue
func (instance *queueHandler) Enqueue(message string, log *logger.LocLoggingEntry) error {
	log.Infof("Putting message to queue '%s'", instance.queueName)
	return instance.queue.Enqueue(message)
}

// Dequeue ... pulls a message from the queue
func (instance *queueHandler) Dequeue(log *logger.LocLoggingEntry) (string, error) {
	log.Infof("Pulling message from queue '%s'", instance.queueName)
	return instance.queue.Dequeue()
}

// NewValueSequence ... creates a new value sequence with the given name (key prefix)
func (instance *coordinator) NewValueSequence(sequenceName string, log *logger.LocLoggingEntry) *valueSequenceHandler {
	log.Infof("Creating new value sequence '%s'", sequenceName)
	sequence := &valueSequenceHandler{
		coordinator: instance,
		keyPrefix:   sequenceName,
	}
	return sequence
}

// AddNew ... adds a new value to the sequence
func (instance *valueSequenceHandler) AddNew(value string, log *logger.LocLoggingEntry) error {
	log.Infof("Adding new value to sequence '%s'", instance.keyPrefix)
	newKey := fmt.Sprintf("%s/%v", instance.keyPrefix, time.Now().UnixNano())
	_, err := (*instance.coordinator).PutIfKeyMissing(newKey, value, log)

	if err != nil {
		log.Errorf("Error posting new sequence value to etcd: %s", err)
	}
	return err
}

// GetAll ... returns the full list of values in the sequence
func (instance *valueSequenceHandler) GetAll(log *logger.LocLoggingEntry) ([]string, error) {
	log.Infof("Getting historical values for sequence '%s'", instance.keyPrefix)
	var result []string
	values, err := (*instance.coordinator).Get(instance.keyPrefix, log, clientv3.WithLimit(0), clientv3.WithPrefix())
	if err != nil {
		log.Errorf("Error retrieving sequence values from etcd: %s", err)
	} else {
		for k := range values {
			result = append(result, values[k].Value)
		}
	}
	return result, err
}

//--private fns
func connect(config Config, log *logger.LocLoggingEntry) (*clientv3.Client, error) {
	var tlsConfig *tls.Config
	if config.Cert == "" {
		tlsConfig = nil
	} else {
		caCert, ioerr := ioutil.ReadFile(config.Cert)
		if ioerr != nil {
			log.WithError(ioerr).Fatalf("Failed to read the certificate file at location %s", config.Cert)
			return nil, ioerr
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig = &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: true,
		}
	}

	response, nrerr := retry(3, 5*time.Second, "ETCD_CONNECT", log,
		func() (interface{}, error) {
			result, err := clientv3.New(clientv3.Config{
				Endpoints:   config.Endpoints,
				DialTimeout: 5 * time.Second,
				Username:    config.Username,
				Password:    config.Password,
				TLS:         tlsConfig,
			})
			return result, err
		},
		func(err error) bool { return true })

	cli, ok := response.(*clientv3.Client)
	if nrerr != nil || !ok {
		log.WithError(nrerr).Errorf("Failed connecting to etcd with endpoints %v , prefix %s, username %s, cert %s", config.Endpoints, config.Prefix, config.Username, config.Cert)
		return nil, nrerr
	}

	basePath := config.Prefix
	cli.KV = namespace.NewKV(cli.KV, basePath)
	cli.Watcher = namespace.NewWatcher(cli.Watcher, basePath)
	cli.Lease = namespace.NewLease(cli.Lease, basePath)

	return cli, nrerr
}

func handleError(err error, log *logger.LocLoggingEntry) bool {
	var retry bool
	if err != nil {
		switch err {
		case context.Canceled:
			log.WithError(err).Error("ctx is canceled by another routine")
		case context.DeadlineExceeded:
			log.WithError(err).Warn("ctx is attached with a deadline is exceeded")
			retry = true
		case rpctypes.ErrEmptyKey:
			log.WithError(err).Error("client-side error")
		default:
			log.WithError(err).Error("bad cluster endpoints, which are not etcd servers")
		}
	}

	return retry
}

//basic retry function that needs to be replaced with exponentatial retry
func retry(attempts int, interval time.Duration, description string, log *logger.LocLoggingEntry, logic func() (interface{}, error), condition func(error) bool) (interface{}, error) {
	var err error
	var result interface{}
	for i := 0; ; i++ {
		result, err := logic()
		//if there was no error or condition to retry was false
		if err == nil || !condition(err) {
			return result, err
		}
		log.WithError(err).Warnf("Failed in attempt, %d / %d for %s. Will retry after sleeping for %d", i+1, attempts, description, interval)
		if i >= (attempts - 1) {
			break
		}
		time.Sleep(interval)
	}
	return result, fmt.Errorf("function %s after %d attempts, last error: %s", description, attempts, err)
}
