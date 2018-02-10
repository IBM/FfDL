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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/trainer/trainer/grpc_trainer_v2"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	sslSuffix = "?ssl=true"
)

// TrainingRecord is the data structure we store in the Mongo collection "trainings"
type TrainingRecord struct {
	ID              bson.ObjectId                    `bson:"_id,omitempty" json:"id"`
	TrainingID      string                           `bson:"training_id" json:"training_id"`
	UserID          string                           `bson:"user_id" json:"user_id"`
	JobID           string                           `bson:"job_id" json:"job_id"`
	ModelDefinition *grpc_trainer_v2.ModelDefinition `bson:"model_definition,omitempty" json:"model_definition"`
	Training        *grpc_trainer_v2.Training        `bson:"training,omitempty" json:"training"`
	Datastores      []*grpc_trainer_v2.Datastore     `bson:"data_stores,omitempty" json:"data_stores"`
	TrainingStatus  *grpc_trainer_v2.TrainingStatus  `bson:"training_status,omitempty" json:"training_status"`
	Metrics         *grpc_trainer_v2.Metrics         `bson:"metrics,omitempty" json:"metrics"`
	Deleted         bool                             `bson:"deleted,omitempty" json:"deleted"`
}

type trainingsRepository struct {
	session    *mgo.Session
	database   string
	collection string
}

type repository interface {
	Store(c *TrainingRecord) error
	Find(trainingID string) (*TrainingRecord, error)
	FindTrainingStatus(trainingID string) (*grpc_trainer_v2.TrainingStatus, error)
	FindTrainingStatusID(trainingID string) (grpc_trainer_v2.Status, error)
	FindTrainingSummaryMetricsString(trainingID string) (string, error)
	FindAll(userID string) ([]*TrainingRecord, error)
	FindCurrentlyRunningTrainings(limit int) ([]*TrainingRecord, error)
	Delete(trainingID string) error
	Close()
}

// ConnectMongo connects to a mongo database collection, using the provided username, password, and certificate file
// It returns a pointer to the session and collection objects, or an error if the connection attempt fails.
// TODO: This function could potentially be moved to a central utility package
func ConnectMongo(mongoURI string, database string, username string, password string, cert string, collection string) (*mgo.Session, *mgo.Collection, error) {

	// See here about the SSL weirdness: https://help.compose.com/docs/connecting-to-mongodb#go--golang-mongodb-and-compose
	uri := strings.TrimSuffix(mongoURI, sslSuffix)
	dialInfo, err := mgo.ParseURL(uri)
	if err != nil {
		log.WithError(err).Errorf("Cannot parse Mongo Connection URI")
		return nil, nil, err
	}
	dialInfo.FailFast = true
	dialInfo.Timeout = 10 * time.Second

	// only do ssl if we have the suffix
	if strings.HasSuffix(mongoURI, sslSuffix) {
		log.Debugf("Using TLS for mongo ...")
		tlsConfig := &tls.Config{}
		roots := x509.NewCertPool()
		if ca, err := ioutil.ReadFile(cert); err == nil {
			roots.AppendCertsFromPEM(ca)
		}
		tlsConfig.RootCAs = roots
		tlsConfig.InsecureSkipVerify = false
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}
	}

	// in case the username/password are not part of the URL string
	if username != "" && password != "" {
		dialInfo.Username = username
		dialInfo.Password = password
	}

	session, err := mgo.DialWithInfo(dialInfo)

	if database == "" {
		database = dialInfo.Database
	}

	if err != nil {
		msg := fmt.Sprintf("Cannot connect to MongoDB at %s, db %s, collection %s", mongoURI, database, collection)
		log.WithError(err).Errorf(msg)
		return nil, nil, err
	}

	collectionObj := session.DB(database).C(collection)

	return session, collectionObj, nil
}

// newTrainingsRepository create a new training repo for storing training data. The mongo URI can contain all the necessary
// connection information. See here: http://docs.mongodb.org/manual/reference/connection-string/
// However, we also support not putting the username/password in the connection URL and provide is separately.
func newTrainingsRepository(mongoURI string, database string, username string, password string, cert string, collection string) (repository, error) {
	log := logger.LocLogger(log.StandardLogger().WithField("module", "trainingRepository"))
	log.Debugf("Creating mongo training repository for %s, collection %s:", mongoURI, collection)

	repo := &trainingsRepository{}

	session, collectionObj, err := ConnectMongo(mongoURI, database, username, password, cert, collection)

	if err != nil {
		return nil, err
	}

	repo.session = session
	repo.collection = collection
	repo.database = collectionObj.Database.Name

	// create index
	collectionObj.EnsureIndexKey("user_id", "training_id")

	return repo, nil
}

func (r *trainingsRepository) Store(t *TrainingRecord) error {
	sess := r.session.Clone()
	defer sess.Close()

	var err error
	if t.ID == "" {
		err = sess.DB(r.database).C(r.collection).Insert(t)
	} else {
		err = sess.DB(r.database).C(r.collection).Update(bson.M{"_id": t.ID}, t)
	}
	if err != nil {
		logWith(t.TrainingID, t.UserID).Errorf("Error storing training record: %s", err.Error())
		return err
	}

	return nil
}

func (r *trainingsRepository) Find(trainingID string) (*TrainingRecord, error) {
	tr := &TrainingRecord{}
	sess := r.session.Clone()
	defer sess.Close()
	err := r.queryDatabase(&bson.M{"training_id": trainingID}, sess).One(tr)
	if err != nil {
		logWithTraining(trainingID).WithError(err).Debugf("Cannot retrieve training record")
		return nil, err
	}
	return tr, nil
}

func (r *trainingsRepository) FindTrainingStatus(trainingID string) (*grpc_trainer_v2.TrainingStatus, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "trainingRepository").WithField(logger.LogkeyTrainingID, trainingID))
	sess := r.session.Clone()
	defer sess.Close()

	tr := &grpc_trainer_v2.TrainingStatus{}
	err := r.queryDatabase(&bson.M{"training_id": trainingID}, sess).Select(bson.M{"TrainingStatus": 1}).One(tr)
	if err != nil {
		logr.WithError(err).Debugf("Cannot retrieve training record")
		return nil, err
	}

	return tr, nil
}

func (r *trainingsRepository) FindTrainingStatusID(trainingID string) (grpc_trainer_v2.Status, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "trainingRepository").WithField(logger.LogkeyTrainingID, trainingID))
	sess := r.session.Clone()
	defer sess.Close()

	tr := &TrainingRecord{}
	err := r.queryDatabase(&bson.M{"training_id": trainingID}, sess).One(tr)
	if err != nil {
		logWithTraining(trainingID).WithError(err).Debugf("Cannot retrieve training record")
		return -1, err
	}

	if tr.TrainingStatus != nil {
		return tr.TrainingStatus.Status, nil
	}
	logr.Debugf("Status not found")
	return grpc_trainer_v2.Status_NOT_STARTED, nil
}

func (r *trainingsRepository) FindTrainingSummaryMetricsString(trainingID string) (string, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "trainingRepository").WithField(logger.LogkeyTrainingID, trainingID))
	sess := r.session.Clone()
	defer sess.Close()

	tr := &TrainingRecord{}
	err := r.queryDatabase(&bson.M{"training_id": trainingID}, sess).One(tr)
	if err != nil {
		logWithTraining(trainingID).WithError(err).Debugf("Cannot retrieve training record")
		return "", err
	}

	if tr.TrainingStatus != nil {
		return tr.Metrics.String(), nil
	}
	logr.Debugf("Status not found")
	return "", nil
}

func (r *trainingsRepository) FindAll(userID string) ([]*TrainingRecord, error) {
	var tr []*TrainingRecord
	sess := r.session.Clone()
	defer sess.Close()

	err := r.queryDatabase(&bson.M{"user_id": userID}, sess).Sort("-training_status.submissiontimestamp").All(&tr)
	if err != nil {
		log.WithField(logger.LogkeyUserID, userID).Errorf("Cannot retrieve all training records: %s", err.Error())
		return nil, err
	}
	return tr, nil
}

func (r *trainingsRepository) Delete(trainingID string) error {
	sess := r.session.Clone()
	defer sess.Close()
	// Perform a soft delete: retain only non-sensitive details of the training record. Note: instead of
	// deleting fields from the record, we upsert with a new record to specify explicitly what should be retained.

	// 1. fetch the record
	existing, err := r.Find(trainingID)
	if err != nil {
		logWithTraining(trainingID).WithError(err).Debugf("Unable to find training record for (soft-)deletion, ID %s: %s", trainingID, err)
		return err
	}
	// 2. update the record
	selector := bson.M{"training_id": trainingID}
	var resources *grpc_trainer_v2.ResourceRequirements
	var status grpc_trainer_v2.TrainingStatus
	var framework *grpc_trainer_v2.Framework
	if existing.Training != nil {
		resources = existing.Training.Resources
	}
	if existing.TrainingStatus != nil {
		status = *existing.TrainingStatus
	}
	if existing.ModelDefinition != nil {
		framework = existing.ModelDefinition.Framework
	}
	newRecord := &TrainingRecord{
		TrainingID: trainingID,
		UserID:     existing.UserID,
		JobID:      existing.JobID,
		ModelDefinition: &grpc_trainer_v2.ModelDefinition{
			Framework: framework,
		},
		Training: &grpc_trainer_v2.Training{
			Resources: resources,
		},
		TrainingStatus: &grpc_trainer_v2.TrainingStatus{
			Status:                 status.Status,
			ErrorCode:              status.ErrorCode,
			SubmissionTimestamp:    status.SubmissionTimestamp,
			CompletionTimestamp:    status.CompletionTimestamp,
			DownloadStartTimestamp: status.DownloadStartTimestamp,
			ProcessStartTimestamp:  status.ProcessStartTimestamp,
			StoreStartTimestamp:    status.StoreStartTimestamp,
		},
		Deleted: true,
	}
	_, err1 := sess.DB(r.database).C(r.collection).Upsert(selector, newRecord)
	if err1 != nil {
		logWithTraining(trainingID).Errorf("Cannot (soft-)delete training record: %s", err.Error())
		return err1
	}
	return nil
}

// queryDatabase serves as the single entry point to run DB queries for this repository. It takes as parameter
// a selector to use for MongoDB's Find(...) method, and returns the query result. Importantly, the method appends
// a "deleted" flag to the query selector to make sure we are never returning records that have been soft-deleted.
func (r *trainingsRepository) queryDatabase(selector *bson.M, sess *mgo.Session) *mgo.Query {
	if selector == nil {
		selector = &bson.M{}
	}
	if (*selector)["deleted"] == nil {
		(*selector)["deleted"] = bson.M{"$ne": true}
	}
	return sess.DB(r.database).C(r.collection).Find(selector)
}

func (r *trainingsRepository) FindCurrentlyRunningTrainings(limit int) ([]*TrainingRecord, error) {
	sess := r.session.Clone()
	defer sess.Close()
	var tr []*TrainingRecord
	//sorting by id in descending fashion(hence the - before id), assumption being records in mongo are being created with auto generated id which has a notion of timestamp built into it
	err := r.queryDatabase(nil, sess).Sort("-_id").Limit(limit).Select(bson.M{"training_status": 1, "training.resources": 2, "training_id": 3}).All(&tr)
	return tr, err
}

func (r *trainingsRepository) Close() {
	log.Debugf("Closing mongo session")
	defer r.session.Close()
}
