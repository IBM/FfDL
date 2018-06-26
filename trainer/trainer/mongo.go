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

	"gopkg.in/mgo.v2"

	log "github.com/sirupsen/logrus"
)

const (
	sslSuffix = "?ssl=true"
)

// ConnectMongo connects to a mongo database collection, using the provided username, password, and certificate file
// It returns a pointer to the session and collection objects, or an error if the connection attempt fails.
// TODO: This function could potentially be moved to a central utility package
func ConnectMongo(mongoURI string, database string, username string, password string, cert string) (*mgo.Session, error) {

	// See here about the SSL weirdness: https://help.compose.com/docs/connecting-to-mongodb#go--golang-mongodb-and-compose
	uri := strings.TrimSuffix(mongoURI, sslSuffix)
	dialInfo, err := mgo.ParseURL(uri)
	if err != nil {
		log.WithError(err).Errorf("Cannot parse Mongo Connection URI")
		return nil, err
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
		msg := fmt.Sprintf("Cannot connect to MongoDB at %s, db %s", mongoURI, database)
		log.WithError(err).Errorf(msg)
		return nil, err
	}

	return session, nil
}
