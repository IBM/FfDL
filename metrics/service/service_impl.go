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

package service

import (
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/service"
	"golang.org/x/net/context"
	tds "github.com/IBM/FfDL/metrics/service/grpc_training_data_v1"
	es "gopkg.in/olivere/elastic.v5"
	"github.com/spf13/viper"
	"encoding/json"
	"crypto/tls"
	"strings"
)

const (
	indexName    = "dlaas_learner_data"

	docTypeLog      = "logline"
	indexMappingLogs = `{
                        "mappings" : {
                            "logline" : {
                                "properties" : {
									"meta" : {
										"properties" : {
											"trainer_id" : { "type" : "keyword", "index" : "not_analyzed" },
											"user_id" : { "type" : "keyword", "index" : "not_analyzed" },
											"time" : { "type" : "long" },
											"rindex" : { "type" : "integer" }
										}
									},
                                    "line" : { "type" : "text", "index" : "not_analyzed" }
                                }
                            }
                        }
                    }`

	docTypeEmetrics      = "emetrics"
	// TODO: How to represent etimes and values maps?  For now, dynamic construction seems to be ok.
	indexMappingEmetrics = `{
                        "mappings" : {
                            "emetrics" : {
                                "properties" : {
									"meta" : {
										"properties" : {
											"trainer_id" : { "type" : "keyword", "index" : "not_analyzed" },
											"user_id" : { "type" : "keyword", "index" : "not_analyzed" },
											"time" : { "type" : "long" },
											"rindex" : { "type" : "integer" }
										}
									},
	                       			"grouplabel" : { "type" : "text", "index" : "not_analyzed" }
                                }
                            }
                        }
                    }`

	elasticSearchAddressKey = "elasticsearch.address"
	elasticSearchUserKey = "elasticsearch.username"
	elasticSearchPwKey = "elasticsearch.password"

	defaultPageSize = 10
)

var (
	// TdsDebugMode = viper.GetBool(TdsDebug)
	TdsDebugMode = false
)

// Service represents the functionality of the training status service
type Service interface {
	tds.TrainingDataServer
	service.LifecycleHandler
}

// TrainingDataService holds the in-memory service context.
type TrainingDataService struct {
	es  *es.Client
	service.Lifecycle
}

func makeDebugLogger(logrr *logrus.Entry, isEnabled bool) *logger.LocLoggingEntry {
	logr := new(logger.LocLoggingEntry)
	logr.Logger = logrr
	logr.Enabled = isEnabled

	return logr
}

// NewService creates a new training status recorder service.
func NewService() Service {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)

	dlogr.Debugf("function entry")

	config.FatalOnAbsentKey(elasticSearchAddressKey)

	elasticSearchAddress := viper.GetString(elasticSearchAddressKey)
	elasticSearchUserName := viper.GetString(elasticSearchUserKey)
	elasticSearchPassworde := viper.GetString(elasticSearchPwKey)

	dlogr.Debugf("elasticSearchAddress: %s", elasticSearchAddress)
	dlogr.Debugf("elasticSearchUserName: %s", elasticSearchUserName)
	dlogr.Debugf("elasticSearchPassworde: %s", elasticSearchPassworde)

	transport := http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify:true,
		},
	}
	client := http.Client{
		Transport: &transport,
	}

	elasticSearchAddresses := strings.Split(elasticSearchAddress, ",")
	for i, v := range elasticSearchAddresses {
		logr.Debugf("es address #%d: %v", i, v)
	}

	esClient, err := es.NewClient(
		es.SetURL(elasticSearchAddresses...),
		es.SetBasicAuth(elasticSearchUserName, elasticSearchPassworde),
		es.SetScheme(viper.GetString("elasticsearch.scheme")),
		es.SetHttpClient(&client),
		es.SetSniff(false),
		es.SetHealthcheck(false),
	)
	if err != nil {
		logr.WithError(err).Errorf("Cannot create elasticsearch client!")
		return nil
	}

	ctx := context.Background() // ?? is ok or no?
	err = createIndexWithLogsIfDoesNotExist(ctx, esClient)
	if err != nil {
		panic(err)
	}

	s := &TrainingDataService{
		es: esClient,
	}
	s.RegisterService = func() {
		tds.RegisterTrainingDataServer(s.Server, s)
	}

	dlogr.Debugf("function exit")
	return s
}

func makeESQueryFromDlaasQuery(in *tds.Query) (es.Query, bool, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)
	var query es.Query

	shouldPostSortProcess := true

	trainingIDFieldName := "meta.training_id.keyword"
	timeFieldName := "meta.time"
	rindexFieldname := "meta.rindex"

	if in == nil || (in.SearchType == tds.Query_ALL) {
		logr.Debugf("Query_ALL")
		query = es.NewMatchAllQuery()
	} else if in.SearchType == tds.Query_NESTED {

		matchQuery := es.NewNestedQuery(
			"meta",
			es.NewMatchQuery(trainingIDFieldName, in.Meta.TrainingId),
		)

		query = matchQuery

	} else if in.SearchType == tds.Query_MATCH {
		logr.Debugf("Query_MATCH")
		bquery := es.NewBoolQuery()

		matchQuery := es.NewNestedQuery(
			"meta",
			es.NewMatchQuery(trainingIDFieldName, in.Meta.TrainingId),
		)

		bquery = bquery.Filter(matchQuery)
		query = bquery
	} else {
		// TODO: This should be a time-based string
		var since int64
		var err error
		if in.Since != "" {
			dlogr.Debugf("Query_ since: %s", in.Since)
			since, err = humanStringToUnixTime(in.Since)
			if err != nil {
				logr.WithError(err).Errorf(
					"For now the since argument must be an integer representing " +
					"the number of milliseconds since midnight January 1, 1970")
				return nil, false, err
			}
		} else if in.Meta.Time != 0 {
			since = in.Meta.Time
		} else {
			since = 0
		}
		if since == 0 {
			if in.Pos > 0 {
				dlogr.Debugf("Query_ NewRangeQuery (pos)")
				query = es.NewBoolQuery().Filter(
					es.NewTermQuery(trainingIDFieldName, in.Meta.TrainingId),
					es.NewBoolQuery().Filter(
						es.NewRangeQuery(rindexFieldname).Gte(in.Pos),
						es.NewRangeQuery(rindexFieldname).Lt(in.Pos + int64(in.Pagesize)),
					),
				)
				shouldPostSortProcess = false
			} else {
				dlogr.Debugf("Query_ NewTermQuery")
				query = es.NewTermQuery(trainingIDFieldName, in.Meta.TrainingId)
			}
		} else {
			dlogr.Debugf("Query_ NewRangeQuery")
			query = es.NewBoolQuery().Filter(
				es.NewTermQuery(trainingIDFieldName, in.Meta.TrainingId),
				es.NewRangeQuery(timeFieldName).Gt(since),
			)
		}
	}
	return query, shouldPostSortProcess, nil
}

// Hello is simple a gRPC test endpoint.
func (c *TrainingDataService) Hello(ctx context.Context, in *tds.Empty) (*tds.HelloResponse, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	logr.Debugf("function entry")
	logr.Debugf("Hello!")
	out := new(tds.HelloResponse)
	out.Msg = "Hello from ffdl-trainingdata-service!"
	logr.Debugf("function exit")
	return out, nil
}

func adjustOffsetPos( pos int ) (int, bool) {
	isBackward := false
	if pos < 0 {
		isBackward = true
		pos = -pos
		if pos >= 1 {
			pos = pos - 1
		}
	}
	return pos, isBackward
}

func (c *TrainingDataService) refreshIndexes(logr *logger.LocLoggingEntry,
	dlogr *logger.LocLoggingEntry) {

	logr.Debugf("Refresh")

	res, err := c.es.Refresh(indexName).Do(context.TODO())
	if err != nil {
		logr.WithError(err).Errorf("Refresh failed")
	}
	if res == nil {
		logr.Errorf("Refresh expected result; got nil")
	}
}

func  (c *TrainingDataService) executeQuery(query es.Query,
	isBackward bool, pos int, pagesize int, typ string, postSortProcess bool,
	dlogr *logger.LocLoggingEntry) (*es.SearchResult, error) {
	var res *es.SearchResult
	var err error
	ctx := context.Background()
	c.refreshIndexes(dlogr, dlogr)
	if postSortProcess {
		dlogr.Debugf("executing query with post-sort processing")
		res, err = c.es.Search(indexName).
			Index(indexName).
			Type(typ).
			Query(query).
			Sort("meta.rindex", !isBackward).
			From(pos).
			Size(pagesize).
			Do(ctx)
	} else {
		dlogr.Debugf("executing query with no post-sort processing")
		res, err = c.es.Search(indexName).
			Index(indexName).
			Type(typ).
			Query(query).
			Sort("meta.rindex", !isBackward).
			Do(ctx)
	}
	return res, err
}

// GetLogs returns a stream of log line records.
func (c *TrainingDataService) GetLogs(in *tds.Query, stream tds.TrainingData_GetLogsServer) (error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))

	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)
	dlogr.Debugf("function entry: %+v", in)

	query, shouldPostSortProcess, err := makeESQueryFromDlaasQuery(in)
	if err != nil {
		logr.WithError(err).Errorf("Could not make elasticsearch query from submitted query: %+v", *in)
		return err
	}
	dlogr.Debugf("pos: %d, since: %s, shouldPostSortProcess: %t", in.Pos, in.Since, shouldPostSortProcess)

	pagesize := int(in.Pagesize)
	if pagesize == 0 {
		pagesize = defaultPageSize
	}

	pos, isBackward := adjustOffsetPos( int(in.Pos) )

	res, err := c.executeQuery(query, isBackward, pos, pagesize, docTypeLog, shouldPostSortProcess, dlogr)

	if err != nil {
		logr.WithError(err).Errorf("Search failed")
		return err
	}

	dlogr.Debugf("Found: %d out of %d", len(res.Hits.Hits), res.TotalHits())

	if !(res.Hits == nil || res.Hits.Hits == nil || len(res.Hits.Hits) == 0) {
		logLineRecord := new(tds.LogLine)
		start := 0
		if isBackward {
			start = len(res.Hits.Hits) - 1
		}
		count := 0

		for i := start; ; {
			err := json.Unmarshal(*res.Hits.Hits[i].Source, &logLineRecord)
			if err != nil {
				logr.WithError(err).Errorf("Unmarshal from ES failed!")
				return err
			}
			err = stream.Send(logLineRecord)
			if err != nil {
				logr.WithError(err).Errorf("stream.Send failed")
				return err
			}

			if isBackward {
				i--
				if i < 0 {
					break
				}
			} else {
				i++
				if i >= len(res.Hits.Hits) {
					break
				}
			}
			count++
			if count >= pagesize {
				break
			}
		}
	}

	dlogr.Debugf("function exit")
	return nil
}

// GetEMetrics returns a stream of evaluation metrics records.
func (c *TrainingDataService) GetEMetrics(in *tds.Query, stream tds.TrainingData_GetEMetricsServer) (error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)
	dlogr.Debugf("function entry: %+v", in)

	query, shouldPostSortProcess, err := makeESQueryFromDlaasQuery(in)
	if err != nil {
		logr.WithError(err).Errorf("Could not make elasticsearch query from submitted query: %+v", *in)
		return err
	}

	pagesize := int(in.Pagesize)
	if pagesize == 0 {
		pagesize = defaultPageSize
	}

	pos, isBackward := adjustOffsetPos( int(in.Pos) )

	dlogr.Debugf("pos: %d, pagesize: %d isBackward: %t", pos, pagesize, isBackward)

	res, err := c.executeQuery(query, isBackward, pos, pagesize, docTypeEmetrics, shouldPostSortProcess, dlogr)

	if err != nil {
		logr.WithError(err).Errorf("Search failed")
		return err
	}

	dlogr.Debugf("Found: %d out of %d", len(res.Hits.Hits), res.TotalHits())

	if !(res.Hits == nil || res.Hits.Hits == nil || len(res.Hits.Hits) == 0) {
		start := 0
		if isBackward {
			start = len(res.Hits.Hits) - 1
		}
		count := 0
		for i := start; ; {
			emetricsRecord := new(tds.EMetrics)
			err := json.Unmarshal(*res.Hits.Hits[i].Source, &emetricsRecord)
			if err != nil {
				logr.WithError(err).Errorf("Unmarshal from ES failed!")
				return err
			}

			err = stream.Send(emetricsRecord)
			if err != nil {
				logr.WithError(err).Errorf("stream.Send failed")
				return err
			}

			// logr.Debugf("EMetrics record: %+v\n", emetricsRecord)
			if isBackward {
				i--
				if i < 0 {
					break
				}
			} else {
				i++
				if i >= len(res.Hits.Hits) {
					break
				}
			}
			count++
			if count >= pagesize {
				break
			}
		}
	}

	dlogr.Debugf("function exit")
	return nil
}

func makeSnippetForDebug(str string, maxLen int) string {
	if TdsDebugMode {
		endEndex := maxLen
		if endEndex >= len(str) {
			endEndex = len(str) - 1
		}
		return str[:endEndex]
	}
	return ""
}

// AddEMetrics adds the passed evaluation metrics record to storage.
func (c *TrainingDataService) AddEMetrics(ctx context.Context, in *tds.EMetrics) (*tds.AddResponse, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))

	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)

	jsonBytes, err := json.Marshal(in)
	if TdsDebugMode {
		dlogr.Debugf("emetrics_in_json: %d: %s",
			in.Meta.Rindex, makeSnippetForDebug(string(jsonBytes), 20))
	}

	_, err = c.es.Index().
		Index(indexName).
		Type(docTypeEmetrics).
		BodyString(string(jsonBytes)).
		Do(ctx)

	out := new(tds.AddResponse)
	if err != nil {
		logr.WithError(err).Errorf("Failed to add log line to elasticsearch")
		out.Success = false
		return out, err
	}

	out.Success = true

	return out, nil
}

// AddLogLine adds the line line record to storage.
func (c *TrainingDataService) AddLogLine(ctx context.Context, in *tds.LogLine) (*tds.AddResponse, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))

	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)
	dlogr.Debugf("AddLogLine: %d: %s", in.Meta.Rindex, makeSnippetForDebug(in.Line, 7))

	out := new(tds.AddResponse)

	_, err := c.es.Index().
		Index(indexName).
		Type(docTypeLog).
		BodyJson(in).
		Do(ctx)

	if err != nil {
		logr.WithError(err).Errorf("Failed to add log line to elasticsearch")
		out.Success = false
		return out, err
	}

	out.Success = true
	dlogr.Debugf("exit")
	return out, nil
}

// DeleteEMetrics deletes the queried evaluation metrics from storage.
func (c *TrainingDataService) DeleteEMetrics(ctx context.Context, in *tds.Query) (*tds.DeleteResponse, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	logr.Debugf("function entry")
	out := new(tds.DeleteResponse)

	query, _, err := makeESQueryFromDlaasQuery(in)
	if err != nil {
		logr.WithError(err).Errorf("Could not make elasticsearch query from submitted query: %+v", *in)
		out.Success = false
		return out, err
	}

	_, err = c.es.DeleteByQuery(indexName).
		Index(indexName).
		Type(docTypeEmetrics).
		Query(query).
		Do(ctx)

	if err != nil {
		logr.WithError(err).Errorf("Search failed failed")
		out.Success = false
		return out, err
	}

	out.Success = true
	logr.Debugf("function exit")
	return out, err
}

// DeleteLogLines deletes the queried log lines from storage.
func (c *TrainingDataService) DeleteLogLines(ctx context.Context, in *tds.Query) (*tds.DeleteResponse, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	logr.Debugf("function entry")
	out := new(tds.DeleteResponse)

	query, _, err := makeESQueryFromDlaasQuery(in)
	if err != nil {
		logr.WithError(err).Errorf("Could not make elasticsearch query from submitted query: %+v", *in)
		out.Success = false
		return out, err
	}

	_, err = c.es.DeleteByQuery(indexName).
		Index(indexName).
		Type(docTypeLog).
		Query(query).
		Do(ctx)

	if err != nil {
		logr.WithError(err).Errorf("Search failed failed")
		out.Success = false
		return out, err
	}

	out.Success = true
	logr.Debugf("function exit")
	return out, err
}

// DeleteJob deletes both queried evaluation metrics and log lines.
func (c *TrainingDataService) DeleteJob(ctx context.Context, in *tds.Query) (*tds.DeleteResponse, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	logr.Debugf("function entry")
	out := new(tds.DeleteResponse)

	query, _, err := makeESQueryFromDlaasQuery(in)
	if err != nil {
		logr.WithError(err).Errorf("Could not make elasticsearch query from submitted query: %+v", *in)
		out.Success = false
		return out, err
	}

	_, err = c.es.DeleteByQuery(indexName).
		Index(indexName).
		Query(query).
		Do(ctx)

	if err != nil {
		logr.WithError(err).Errorf("Search failed failed")
		out.Success = false
		return out, err
	}

	out.Success = true
	logr.Debugf("function exit")
	return out, err
}


// ======================================

func createSubIndexIfDoesNotExist(ctx context.Context, client *es.Client,
	subIndex string,  logr *logger.LocLoggingEntry) error {
	logr.Debugf("calling CreateIndex")
	res, err := client.CreateIndex(indexName).
		Body(subIndex).
		Do(ctx)

	if err != nil {
		logr.WithError(err).Debug("CreateIndex failed")
		return err
	}
	if !res.Acknowledged {
		return errors.New("the creation of index was not acknowledged")
	}
	return nil
}

func createIndexWithLogsIfDoesNotExist(ctx context.Context, client *es.Client) error {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	logr.Debugf("function entry")

	logr.Debugf("calling IndexExists")
	exists, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		logr.WithError(err).Debug("IndexExists failed")
		return err
	}

	if exists {
		return nil
	}

	err = createSubIndexIfDoesNotExist(ctx, client, indexMappingLogs,  logr)
	if err != nil {
		logr.WithError(err).Debug("CreateIndex for logs failed")
		return err
	}
	err = createSubIndexIfDoesNotExist(ctx, client, indexMappingEmetrics,  logr)
	if err != nil {
		logr.WithError(err).Debug("CreateIndex for logs failed")
		return err
	}

	logr.Debugf("function exit")
	return err
}

