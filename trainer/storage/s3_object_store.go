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

package storage

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/IBM/FfDL/commons/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/spf13/viper"
	"github.com/IBM/FfDL/commons/logger"
)

const (
	// DataStoreTypeS3 is the type string for the S3-based object store.
	DataStoreTypeMountVolume = "mount_volume"
	DataStoreTypeMountCOSS3 = "mount_cos"
	DataStoreTypeS3 = "s3_datastore"
	defaultRegion   = "us-standard"
)

type s3ObjectStore struct {
	conf    map[string]string
	client  *s3.S3
	session *session.Session
}

// NewS3ObjectStore creates a new connector for the IBM S3 based object store (https://ibm-public-cos.github.io).
// If `session` is set to nil, the configuration is read via viper. If conn is provided, it will use it for connecting
// to the provided object store.
func NewS3ObjectStore(conf map[string]string) (DataStore, error) {
	if conf == nil {
		return nil, fmt.Errorf("conf argument is nil")
	}

	// check config and fatal if not found
	config.FatalOnAbsentKeyInMap(AuthURLKey, conf)
	config.FatalOnAbsentKeyInMap(UsernameKey, conf)
	config.FatalOnAbsentKeyInMap(PasswordKey, conf)

	return &s3ObjectStore{
		conf: conf,
	}, nil
}

func (os *s3ObjectStore) Connect() error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))

	cfg := aws.NewConfig().
		WithEndpoint(os.conf[AuthURLKey]).
		WithRegion(os.getRegion()).
		WithS3ForcePathStyle(os.usePathStyleAddressing()).
		WithCredentials(credentials.NewStaticCredentials(
			os.conf[UsernameKey],
			os.conf[PasswordKey], "")).
		WithMaxRetries(10)

	sess, err := session.NewSession(cfg)
	if err != nil {
		logr.WithError(err).Error("Error creating new S3 session. Please check the config or connectivity.")
		return err
	}
	os.session = sess

	// make deployment outside SL work for local env
	replaceS3ObjectStoreURL(os.session)
	os.conf[AuthURLKey] = *os.session.Config.Endpoint
	os.client = s3.New(os.session)
	return nil
}

// Determine whether the S3 client should use path-style addressing or not
// See http://docs.aws.amazon.com/AmazonS3/latest/dev/VirtualHosting.html
func (os *s3ObjectStore) usePathStyleAddressing() bool {
	// to enable running in local environment (minikube), use s3 path style
	// addressing if the URL containts a kubernetes host name.
	if strings.Contains(os.conf[AuthURLKey], "svc.cluster.local") {
		return true
	}
	// by default, don't use path-style addressing (i.e., use domain name based addressing)
	return false
}

func (os *s3ObjectStore) UploadArchive(container string, object string, payload []byte) error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if os.client == nil {
		return ErrNotConnected
	}
	exists, err := os.ContainerExists(container)
	if err != nil {
		logr.WithError(err).Errorf("Error checking if container '%s' exists", container)
		return err
	}
	if !exists { // create bucket
		err = os.createBucket(container)
		if err != nil {
			logr.WithError(err).Errorf("Error creating bucket '%t'", exists)
			return err
		}
	}

	uploader := s3manager.NewUploader(os.session)
	upParams := &s3manager.UploadInput{
		Bucket: aws.String(container),
		Key:    aws.String(object),
		Body:   bytes.NewReader(payload),
	}
	_, err = uploader.Upload(upParams)
	if err != nil {
		logr.Errorf("Uploading archive %s/%s failed: %s", container, object, err.Error())
		return err
	}
	return nil
}

func (os *s3ObjectStore) DownloadArchive(container string, object string) ([]byte, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if os.client == nil {
		return nil, ErrNotConnected
	}
	downloader := s3manager.NewDownloader(os.session)

	input := &s3.GetObjectInput{
		Bucket: aws.String(container),
		Key:    aws.String(object),
	}
	var payload []byte
	buff := aws.NewWriteAtBuffer(payload)
	_, err := downloader.Download(buff, input)
	if err != nil {
		logr.Errorf("Downloading archive %s/%s failed: %s", container, object, err.Error())
		return nil, err
	}
	return buff.Bytes(), nil
}

func (os *s3ObjectStore) DeleteArchive(container string, object string) error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if os.client == nil {
		return ErrNotConnected
	}
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(container),
		Key:    aws.String(object),
	}
	_, err := os.client.DeleteObject(input)
	if err != nil {
		logr.Errorf("Deleting archive %s/%s failed: %s", container, object, err.Error())
		return err
	}

	// check if no more
	resp, err := os.client.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(container),
	})
	if err != nil {
		logr.Errorf("Listing objects in %s failed: %s", container, err.Error())
		return err
	}
	if len(resp.Contents) == 0 {
		if err := os.deleteBucket(container); err != nil {
			logr.Errorf("Deleting bucket in %s failed: %s", container, err.Error())
			return err
		}
	}
	return nil
}

func (os *s3ObjectStore) GetTrainedModelSize(path string, numLearners int32) (int64, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if os.client == nil {
		return 0, ErrNotConnected
	}

	// set default numLearners if not provided
	if numLearners <= 0 {
		numLearners = 1
	}

	// separate path in container and object
	var container, objectPrefix string
	index := strings.Index(path, "/")
	if index < 0 {
		container = path
	} else {
		container = path[0:index]
		objectPrefix = path[index+1:]
	}

	var trainedModelSize int64

	//Iterate over all learner objects
	for iter := 0; iter <= int(numLearners); iter++ {
		var pathToProcess string
		if iter == 0 { // gather root folder contents too
			pathToProcess = objectPrefix + "/"
		} else {
			pathToProcess = objectPrefix + "/learner-" + strconv.Itoa(iter) + "/"
		}

		resp, err := os.client.ListObjects(&s3.ListObjectsInput{
			Bucket:    aws.String(container),
			Delimiter: aws.String("/"),
			Prefix:    aws.String(pathToProcess),
		})

		if err != nil {
			logr.Errorf("Listing objects in bucket %s failed: %s", container, err.Error())
			return 0, err
		}

		for _, obj := range resp.Contents {
			logr.Debugf("trained model object: %s, %d", *obj.Key, *obj.Size)
			trainedModelSize += *obj.Size
		}
	}
	return trainedModelSize, nil

}

func (os *s3ObjectStore) copyDirToZipStream(container string, path string, w *zip.Writer, recursive bool,
	log *logger.LocLoggingEntry) error {

	delimiter := "/"
	if recursive {
		// If recursive we do not delimit.
		delimiter = ""
	}
	log.Debugf("containing string is: %s", container)
	resp, err := os.client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:    aws.String(container),
		Delimiter: aws.String(delimiter),
		Prefix:    aws.String(path),
	})

	// log.Debugf("objects: %s", resp.Contents)
	if err != nil {
		log.Errorf("Listing objects in bucket %s failed: %s", container, err.Error())
		return err
	}

	for _, obj := range resp.Contents {
		log.Debugf("Downloading trained model object: %s, %d", *obj.Key, *obj.Size)

		f, err := w.Create(*obj.Key)
		if err != nil {
			log.Errorf("Creating zip entry %s failed: %s", *obj.Key, err.Error())
			return err
		}

		input := &s3.GetObjectInput{
			Bucket: aws.String(container),
			Key:    aws.String(*obj.Key),
		}

		// TODO download hangs
		// TODO try to do this without a buffer (need a way to bridge io.Writer and io.WriterAt)
		// payload := make([]byte, 1024)
		// buff := aws.NewWriteAtBuffer(make([]byte, 1024*1024))
		// numBytes, err := downloader.Download(buff, input)
		// log.Debugf("number of bytes read: ", numBytes)
		// if err != nil {
		// 	log.Errorf("Downloading object %s/%s failed: %s", container, *obj.Key, err.Error())
		// 	return err
		// }
		result, err := os.client.GetObject(input)
		if err != nil {
			log.Errorf("Cannot get object from datastore: %s", err.Error())
			return err
		}

		_, err = io.Copy(f, result.Body) // FIXME this is not efficient
		if err != nil {
			log.Errorf("Cannot write to zip stream: %s", err.Error())
			return err
		}
	}
	return nil
}

func (os *s3ObjectStore) DownloadTrainedModelAsZipStream(path string, numLearners int32, writer io.Writer) error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if os.client == nil {
		return ErrNotConnected
	}

	// set default numLearners if not provided
	if numLearners <= 0 {
		numLearners = 1
	}

	// separate path in container and object
	var container, objectPrefix string
	index := strings.Index(path, "/")
	if index < 0 {
		container = path
	} else {
		container = path[0:index]
		objectPrefix = path[index+1:]
	}
	if objectPrefix != "" {
		// Best effort to get the training ID
		logr = logr.WithField(logger.LogkeyTrainingID, objectPrefix)
	}
	logr.Debugf("Number of learners is: %d", numLearners)

	// Create a buffer to write our zip to
	w := zip.NewWriter(writer)

	// Download the root folder contents (model files)
	pathToRootFolder := objectPrefix + "/"
	resp, err := os.client.ListObjects(&s3.ListObjectsInput{
		Bucket:    aws.String(container),
		Delimiter: aws.String("/"),
		Prefix:    aws.String(pathToRootFolder),
	})
	// logr.Debugf("objects: %s", resp.Contents)
	if err != nil {
		logr.Errorf("Listing objects in bucket %s failed: %s", container, err.Error())
		return err
	}

	for _, obj := range resp.Contents {
		logr.Debugf("Downloading trained model object: %s, %d", *obj.Key, *obj.Size)

		f, err := w.Create(*obj.Key)
		if err != nil {
			logr.Errorf("Creating zip entry %s failed: %s", *obj.Key, err.Error())
			return err
		}

		input := &s3.GetObjectInput{
			Bucket: aws.String(container),
			Key:    aws.String(*obj.Key),
		}

		result, err := os.client.GetObject(input)
		if err != nil {
			logr.Errorf("Cannot get object from datastore: %s", err.Error())
			return err
		}

		_, err = io.Copy(f, result.Body) // FIXME this is not efficient
		if err != nil {
			logr.Errorf("Cannot write to zip stream: %s", err.Error())
			return err
		}
	}

	//Download learner folder contents
	//Iterate over all learner objects
	for iter := 0; iter < int(numLearners); iter++ {
		pathToLearner := objectPrefix + "/learner-" + strconv.Itoa(iter+1) + "/"

		err := os.copyDirToZipStream(container, pathToLearner, w, true, logr)

		if err != nil {
			logr.WithError(err).Error("Copy of learner contents failed!")
			return err
		}
	}

	pathToLogs := objectPrefix + "/logs/"

	err = os.copyDirToZipStream(container, pathToLogs, w, true, logr)

	if err != nil {
		log.WithError(err).Error("Copy of logs contents failed!")
		return err
	}

	if err := w.Close(); err != nil {
		logr.Errorf("Cannot close zip stream: %s", err.Error())
		return err
	}
	return nil
}

func (os *s3ObjectStore) DownloadTrainedModelLogFile(path string, numLearners int32, learnerIndex int32, objectPath string, writer io.Writer) error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))

	logr.Debugf("s3ObjectStore DownloadTrainedModelLogFile: bucket: %s, key: %s", path, objectPath)
	if os.client == nil {
		return ErrNotConnected
	}

	// separate path in container and object
	var container, objectPrefix string
	index := strings.Index(path, "/")
	if index < 0 {
		container = path
	} else {
		container = path[0:index]
		objectPrefix = path[index+1:]
	}
	if objectPrefix != "" {
		// Best effort to get the training ID
		logr = logr.WithField(logger.LogkeyTrainingID, objectPrefix)
	}
	logr.Debugf("Number of learners is: %d", numLearners)

	var nbytesToProcess int64
	var buff *aws.WriteAtBuffer
	learnerPath := "/learner-" + strconv.Itoa(int(learnerIndex))
	for i := 0; i < 2; i++ {

		fullObjectPath := objectPrefix + learnerPath + "/" + objectPath

		logr.Debugf("DownloadTrainedModelLogFile: container: %s, path: %s", container, fullObjectPath)

		downloader := s3manager.NewDownloader(os.session)
		input := &s3.GetObjectInput{
			Bucket: aws.String(container),
			Key:    aws.String(fullObjectPath),
		}
		var payload []byte
		buff = aws.NewWriteAtBuffer(payload)
		var err error
		nbytesToProcess, err = downloader.Download(buff, input)
		if err != nil {
			if i == 0 {
				logr.Debug("First attempt to download failed, trying with no learner path")
				learnerPath = ""
				continue
			} else {
				logr.WithError(err).Errorf("Downloading log file %s failed", fullObjectPath)
				return err
			}
		} else {
			break
		}
	}
	posStart := int64(0)

	for posStart < nbytesToProcess {
		// var nWritten int
		nWritten, err := writer.Write(buff.Bytes()[posStart:])
		posStart += int64(nWritten)
		if posStart < nbytesToProcess {
			// I can't so far confirm this is ever happening
			logr.Debugf("Write chunked %d total, %d written", nbytesToProcess, posStart)
		}
		if err != nil {
			logr.WithError(err).Errorf("Cannot write to zip stream")
			return err
		}
	}
	return nil
}

func (os *s3ObjectStore) ContainerExists(name string) (bool, error) {
	if os.client == nil {
		return false, ErrNotConnected
	}

	if strings.Contains(name, "/") {
		return false, fmt.Errorf("bucket name contains an illegal character")
	}

	params := &s3.HeadBucketInput{
		Bucket: aws.String(name),
	}
	_, err := os.client.HeadBucket(params)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NotFound" {
				return false, nil
			}
		}
	}
	return err == nil, err
}

// Disconnect destroys the S3 client and the associated session.
func (os *s3ObjectStore) Disconnect() {
	os.client = nil
	os.session = nil
}

func (os *s3ObjectStore) createBucket(name string) error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	params := &s3.CreateBucketInput{
		Bucket: aws.String(name),
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{
			LocationConstraint: aws.String(os.getRegion()),
		},
	}
	_, err := os.client.CreateBucket(params)
	if err != nil {
		logr.Errorf("Creating bucket failed: %s", err.Error())
		return err
	}
	return nil
}

func (os *s3ObjectStore) deleteBucket(name string) error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	params := &s3.DeleteBucketInput{
		Bucket: aws.String(name),
	}
	_, err := os.client.DeleteBucket(params)
	if err != nil {
		logr.Errorf("Deleting bucket failed: %s", err.Error())
	}
	return nil
}

// getRegion reads the viper config for a region, otherwise it will use a default.
func (os *s3ObjectStore) getRegion() string {
	if region, ok := os.conf[RegionKey]; ok {
		return region
	}
	return defaultRegion
}

// replaceS3ObjectStoreURL depending on the DLAAS_ENV variables we need to
// "rewrite" the object store URL to either use the internal SL proxy or the
// external URLs.
func replaceS3ObjectStoreURL(sess *session.Session) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	env := viper.GetString(config.EnvKey)

	switch env {
	case config.DevelopmentEnv:
		replaceS3ConnURL(sess, intDevBMObjectStoreURL)
	case config.StagingEnv:
		replaceS3ConnURL(sess, intStagingBMObjectStoreURL)
	case config.ProductionEnv:
		replaceS3ConnURL(sess, intProdBMObjectStoreURL)
	default: // any other env will use local settings (assuming outside SL)
		if strings.Contains(*sess.Config.Endpoint, intSLObjectStoreURLFragment) {
			newURL := strings.Replace(*sess.Config.Endpoint, intSLObjectStoreURLFragment, extSLObjectStoreURLFragment, 1)
			logr.Debugf("Replaced ObjectStore URL from %s to %s", *sess.Config.Endpoint, newURL)
			sess.Config.Endpoint = &newURL
		} else if strings.Contains(*sess.Config.Endpoint, intDevBMObjectStoreURL) ||
			strings.Contains(*sess.Config.Endpoint, intStagingBMObjectStoreURL) ||
			strings.Contains(*sess.Config.Endpoint, intProdBMObjectStoreURL) {
			logr.Debugf("Replaced ObjectStore URL from %s to %s", *sess.Config.Endpoint, extBMObjectStoreURL)
			*sess.Config.Endpoint = extBMObjectStoreURL
		}
	}
}

func replaceS3ConnURL(sess *session.Session, bluemixOSURL string) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if strings.Contains(*sess.Config.Endpoint, extSLObjectStoreURLFragment) { // external SL URL part
		newURL := strings.Replace(*sess.Config.Endpoint, extSLObjectStoreURLFragment, intSLObjectStoreURLFragment, 1)
		logr.Debugf("Replaced ObjectStore URL from %s to %s", *sess.Config.Endpoint, newURL)
		sess.Config.Endpoint = &newURL
	} else if strings.Contains(*sess.Config.Endpoint, extBMObjectStoreURL) { // external BM
		logr.Debugf("Replaced ObjectStore URL from %s to %s", *sess.Config.Endpoint, bluemixOSURL)
		sess.Config.Endpoint = &bluemixOSURL
	}
}
