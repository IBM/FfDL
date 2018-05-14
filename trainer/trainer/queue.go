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

package trainer

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/IBM/FfDL/commons/logger"

	"github.com/ventu-io/go-shortid"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	lockID = "queue_lock"

	lockRetries    = 15 // iterations
	lockRetryDelay = 5  // seconds
	lockExpiration = 60 // seconds
)

// JobQueue represents the functionality of a queue
type JobQueue interface {
	Enqueue(string) error
	Dequeue() (string, error)
	Peek() (string, error)
	Delete(string) (bool, error)
	Size() (int, error)
	Empty() (bool, error)
	Lock() error
	Unlock() error
}

// TrainingJobQueue is a JobQueue backed by mongo
type TrainingJobQueue struct {
	queue           []Entry
	queueID         string // unique identifier for this queue so we can acquire lock
	session         *mgo.Session
	database        string
	queueCollection string
	lockCollection  string
}

// Entry represents a single training job in the queue
type Entry struct {
	ID         bson.ObjectId `bson:"_id" json:"id"`
	TrainingID string        `bson:"training_id" json:"training_id"`
	Submitted  time.Time     `bson:"submitted" json:"submitted"`
}

// LockEntry represents which trainer service currently has a lock on the queue
type LockEntry struct {
	ID      bson.ObjectId `bson:"_id" json:"id"`
	LockID  string        `bson:"lock" json:"lock"`
	Owner   string        `bson:"owner" json:"owner"`
	Expires time.Time     `bson:"expires" json:"expires"`
}

// newTrainingJobQueue creates a new queue for training jobs
func newTrainingJobQueue(mongoURI string, database string, username string, password string, cert string, queueCollection string, lockCollection string) (*TrainingJobQueue, error) {
	log := logger.LocLogger(log.StandardLogger().WithField("module", "trainingQueue"))
	log.Debugf("Creating mongo queue repository for %s", mongoURI)

	session, err := ConnectMongo(mongoURI, database, username, password, cert)
	if err != nil {
		return nil, err
	}

	// ensure lock id is unique
	sess := session.Clone()
	defer sess.Close()
	sess.DB(database).C(lockCollection).EnsureIndex(mgo.Index{
		Key:    []string{"name", "lock"},
		Unique: true,
	})

	sid, _ := shortid.Generate()
	q := &TrainingJobQueue{
		queue:           make([]Entry, 0),
		queueID:         fmt.Sprintf("queue-%s", sid),
		session:         session,
		database:        database,
		queueCollection: queueCollection,
		lockCollection:  lockCollection,
	}
	return q, nil
}

// Enqueue adds a training job id to the queue
// trainer should acquire the lock before calling Enqueue()
func (q *TrainingJobQueue) Enqueue(id string) error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "trainingQueue").WithField(logger.LogkeyTrainingID, id))

	sess := q.session.Clone()
	defer sess.Close()

	entry := &Entry{
		ID:         bson.NewObjectId(),
		TrainingID: id,
		Submitted:  time.Now(),
	}
	err := sess.DB(q.database).C(q.queueCollection).Insert(entry)
	if err != nil {
		logr.WithError(err).Errorf("Failed to queue training job %s", id)
		return err
	}
	return nil
}

// Dequeue returns a single training job id and removes it from the queue
// trainer should acquire the lock before calling Dequeue()
func (q *TrainingJobQueue) Dequeue() (string, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "trainingQueue"))

	sess := q.session.Clone()
	defer sess.Close()

	// find the earliest entry
	entry := &Entry{}
	err := sess.DB(q.database).C(q.queueCollection).Find(nil).Sort("submitted").One(&entry)

	if err == mgo.ErrNotFound {
		logr.Debugf("queue is empty")
		return "", fmt.Errorf("queue is empty")
	}

	err = sess.DB(q.database).C(q.queueCollection).Remove(bson.M{"_id": entry.ID})
	if err != nil {
		logr.WithError(err).Errorf("failed to delete %+v from queue", entry)
		return "", err
	}

	return entry.TrainingID, nil
}

// Peek returns a single training job id and leaves it in the queue
func (q *TrainingJobQueue) Peek() (string, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "trainingQueue"))

	sess := q.session.Clone()
	defer sess.Close()

	// find the earliest entry
	entry := &Entry{}
	err := sess.DB(q.database).C(q.queueCollection).Find(nil).Sort("submitted").One(&entry)

	if err == mgo.ErrNotFound {
		logr.Debugf("queue is empty")
		return "", fmt.Errorf("queue is empty")
	}

	return entry.TrainingID, nil
}

// Delete removes a training job id from any position in the queue
// trainer should acquire the lock before calling Delete()
func (q *TrainingJobQueue) Delete(id string) (bool, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "trainingQueue").WithField(logger.LogkeyTrainingID, id))

	sess := q.session.Clone()
	defer sess.Close()

	entry := &Entry{}
	err := sess.DB(q.database).C(q.queueCollection).Find(bson.M{"training_id": id}).One(&entry)

	if err == mgo.ErrNotFound {
		logr.Debugf("%s not found", id)
		return false, nil
	}

	err = sess.DB(q.database).C(q.queueCollection).Remove(bson.M{"_id": entry.ID})
	if err != nil {
		logr.WithError(err).Errorf("failed to delete %+v from queue", entry)
		return false, err
	}

	return true, nil
}

// Size returns the number of elements in the queue.
func (q *TrainingJobQueue) Size() (int, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "trainingQueue"))

	sess := q.session.Clone()
	defer sess.Close()

	count, err := sess.DB(q.database).C(q.queueCollection).Count()
	if err != nil {
		logr.WithError(err).Errorf("failed to check queue size: %s", err.Error())
		return 0, err
	}
	return count, nil
}

// Empty returns whether the queue has any jobs
func (q *TrainingJobQueue) Empty() (bool, error) {
	size, err := q.Size()
	return size == 0, err
}

// Lock acquires a distributed lock in mongo
// trainer should use this when pulling jobs, so that multiple trainers do not peek/submit the same job to lcm
func (q *TrainingJobQueue) Lock() error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "trainingQueue"))

	sess := q.session.Clone()
	defer sess.Close()

	var lockErr error
	for i := 0; i < lockRetries; i++ {
		// remove expired locks
		sess.DB(q.database).C(q.lockCollection).RemoveAll(bson.M{"expires": bson.M{"$lt": time.Now()}})

		// write lockID, processID to mongo
		lockEntry := &LockEntry{
			ID:      bson.NewObjectId(),
			LockID:  lockID,
			Owner:   q.queueID,
			Expires: time.Now().Add(lockExpiration * time.Second),
		}
		lockErr = sess.DB(q.database).C(q.lockCollection).Insert(lockEntry)

		// if write succeeds, lock is acquired
		if lockErr == nil {
			return nil
		}

		time.Sleep(lockRetryDelay * time.Second)
	}
	logr.WithError(lockErr).Warnf("failed to acquire lock: %s", lockErr.Error())
	return lockErr
}

// Unlock releases the lock in mongo
func (q *TrainingJobQueue) Unlock() error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "trainingQueue"))

	sess := q.session.Clone()
	defer sess.Close()

	// remove matching lock id and owner
	err := sess.DB(q.database).C(q.lockCollection).Remove(bson.M{"lock": lockID, "owner": q.queueID})
	if err != nil {
		logr.WithError(err).Warnf("Failed to remove lock: %s", err.Error())
		return err
	}

	return nil
}

// QueueName returns the name of the queue collection in mongo based on the GPU type
func QueueName(gpuType string) string {
	return "TRAINING_JOB_QUEUE_" + TransformResourceName(gpuType)
}

// LockName returns the name of the lock collection in mongo based on the GPU type
func LockName(gpuType string) string {
	return "LOCK_" + TransformResourceName(gpuType)
}

//TransformResourceName performs replacement and capitalization so resource names are consistent
func TransformResourceName(resource string) string {
	return strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(resource))
}
