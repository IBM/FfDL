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
	"fmt"
	"time"
)

const (
	indexName          = "tds_index"  // New alias
	indexNameV1       = "dlaas_learner_data"
	indexNameV2       = "dlaas_learner_data_v2"

	//"time" : { "type": "date", "format": "epoch_millis" },
	metaSubRecord = `"meta" : {
		"properties" : {
			"trainer_id" : { "type" : "keyword", "index" : "not_analyzed" },
			"user_id" : { "type" : "keyword", "index" : "not_analyzed" },
			"time" : { "type" : "long" },
			"rindex" : { "type" : "integer" },
			"subid" : { "type" : "keyword", "null_value" : "NULL", "index" : "not_analyzed" }
		}
	}`

	docTypeLog      = "logline"
	indexMappingLogs = `{
                            "logline" : {
                                "properties" : {
									` + metaSubRecord + `,
                                    "line" : { "type" : "text", "index" : "not_analyzed" }
                                }
                            }
                        }`

	docTypeEmetrics      = "emetrics"
	indexMappingEmetrics = `{
                            "emetrics" : {
                                "properties" : {
									` + metaSubRecord + `,
	                       			"grouplabel" : { "type" : "text", "index" : "not_analyzed" }
                                }
                            }
                        }`
	// TODO: How to represent etimes and values maps?  For now, dynamic construction seems to be ok.

	elasticSearchAddressKey = "elasticsearch.address"
	elasticSearchUserKey = "elasticsearch.username"
	elasticSearchPwKey = "elasticsearch.password"

	defaultPageSize = 10

)

var (
	// TdsDebugMode = viper.GetBool(TdsDebug)
	TdsDebugMode = false

	// TdsDebugLogLineAdd outputs diagnostic loglines as they are added.  Should normally be false.
	TdsDebugLogLineAdd = false

	// TdsDebugEMetricAdd outputs diagnostic emetrics as they are added.  Should normally be false.
	TdsDebugEMetricAdd = false

	// TdsReportTimes if try report timings for Elastic Search operations
	TdsReportTimes = false
)

// Service represents the functionality of the training status service
type Service interface {
	tds.TrainingDataServer
	service.LifecycleHandler
}

// TrainingDataService holds the in-memory service context.
type TrainingDataService struct {
	es  *es.Client
	esBulkProcessor *es.BulkProcessor
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
	//noinspection GoBoolExpressions
	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)

	dlogr.Debugf("function entry")

	config.FatalOnAbsentKey(elasticSearchAddressKey)

	elasticSearchAddress := viper.GetString(elasticSearchAddressKey)
	elasticSearchUserName := viper.GetString(elasticSearchUserKey)
	elasticSearchPassword := viper.GetString(elasticSearchPwKey)

	dlogr.Debugf("elasticSearchAddress: %s", elasticSearchAddress)
	dlogr.Debugf("elasticSearchUserName: %s", elasticSearchUserName)
	dlogr.Debugf("elasticSearchPassword: %s", elasticSearchPassword)

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
		es.SetBasicAuth(elasticSearchUserName, elasticSearchPassword),
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

	//// Setup a bulk processor
	//esBulkProcessor, err := esClient.BulkProcessor().
	//	Name("MyBackgroundWorker-1").
	//	Workers(2).
	//	BulkActions(10).              // commit if # requests >= 1000
	//	BulkSize(1 << 20).              // commit if size of requests >= 1 MB
	//	FlushInterval(30 * time.Second).  // commit every 30s
	//	Do(context.Background())
	//if err != nil {
	//	panic(err)
	//}

	s := &TrainingDataService{
		es: esClient,
		//esBulkProcessor: esBulkProcessor,
	}
	s.RegisterService = func() {
		tds.RegisterTrainingDataServer(s.Server, s)
	}

	dlogr.Debugf("function exit")
	return s
}

func makeESQueryFromDlaasQuery(in *tds.Query) (es.Query, bool, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	//noinspection GoBoolExpressions
	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)
	var query es.Query

	shouldPostSortProcess := true

	trainingIDFieldName := "meta.training_id.keyword"
	timeFieldName := "meta.time"
	rindexFieldName := "meta.rindex"
	subidFieldName := "meta.subid"

	if in.SearchType == tds.Query_TERM {
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
		var idQuery es.Query
		if in.Meta.Subid != "" {
			idQuery = es.NewBoolQuery().Filter(
				es.NewTermQuery(trainingIDFieldName, in.Meta.TrainingId),
				es.NewTermQuery(subidFieldName, in.Meta.Subid),
			)
		} else {
			idQuery = es.NewTermQuery(trainingIDFieldName, in.Meta.TrainingId)
		}

		if since == 0 {
			if in.Pos > 0 {
				dlogr.Debugf("Query_ NewRangeQuery (pos)")
				query = es.NewBoolQuery().Filter(
					idQuery,
					es.NewBoolQuery().Filter(
						es.NewRangeQuery(rindexFieldName).Gte(in.Pos),
						es.NewRangeQuery(rindexFieldName).Lt(in.Pos + int64(in.Pagesize)),
					),
				)
				shouldPostSortProcess = false
			} else {
				dlogr.Debugf("Query_ NewTermQuery")
				query = idQuery
			}
		} else {
			dlogr.Debugf("Query_ NewRangeQuery")
			query = es.NewBoolQuery().Filter(
				idQuery,
				es.NewRangeQuery(timeFieldName).Gte(since),
			)
		}
	} else {
		err := fmt.Errorf("search type not supported: %s", tds.Query_SearchType_name[int32(in.SearchType)])
		logr.WithError(err).Error("Can't perform query")
		return nil, false, err
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

func (c *TrainingDataService) refreshIndexes(ctx context.Context, logr *logger.LocLoggingEntry) {

	logr.Debugf("Refresh")

	res, err := c.es.Refresh(indexName).Do(ctx)
	if err != nil {
		logr.WithError(err).Debugf("Refresh failed")
	}
	if res == nil {
		logr.Debugf("Refresh expected result; got nil")
	}
	c.es.Update()
}

//noinspection GoBoolExpressions
func (c *TrainingDataService) reportTime(logr *logger.LocLoggingEntry,
	method string,
	trainingID string,
	rIndex int64,
	logReportTimeInMilliseconds int64,
	endPointEntry time.Time, doneQuery time.Time,
) {
	if TdsReportTimes {
		logReportTime := time.Unix(0, logReportTimeInMilliseconds*int64(time.Millisecond))

		elapsedFromReportTimeToEndPointEntryTime := endPointEntry.Sub(logReportTime)

		elapsedFromEndPointEntryToQueryDone := doneQuery.Sub(endPointEntry)

		elapsedDoneQueryToExit := time.Since(doneQuery)

		//logr.Debugf("%s query took %f", method, elapsed.Seconds())
		fmt.Printf("%7d\t%12.3f\t%12.3f\t%12.3f\t%s\t%s\n",
			int(rIndex),
			elapsedFromReportTimeToEndPointEntryTime.Seconds(),
			elapsedFromEndPointEntryToQueryDone.Seconds(),
			elapsedDoneQueryToExit.Seconds(),
			method, trainingID)
	}
}

func  (c *TrainingDataService) executeQuery(index string, query es.Query,
	isBackward bool, pos int, pagesize int, typ string, postSortProcess bool,
	dlogr *logger.LocLoggingEntry) (*es.SearchResult, error) {
	var res *es.SearchResult
	var err error
	ctx := context.Background()

	//c.refreshIndexes(dlogr)

	if postSortProcess {
		dlogr.Debugf("executing query with post-sort processing")
		res, err = c.es.Search(index).
			Index(index).
			Type(typ).
			Query(query).
			Sort("meta.time", !isBackward).
			From(pos).
			Size(pagesize).
			Do(ctx)
	} else {
		dlogr.Debugf("executing query with no post-sort processing")
		res, err = c.es.Search(index).
			Index(index).
			Type(typ).
			Query(query).
			Sort("meta.time", !isBackward).
			Do(ctx)
	}

	return res, err
}

func (c *TrainingDataService) reportOnCluster(method string, logr *logger.LocLoggingEntry) {
	//ctx := context.Background()
	//
	//res, err := c.es.ClusterHealth().Do(ctx)
	//if err != nil {
	//	logr.WithError(err).Errorf("can't get cluster health!")
	//	return
	//}
	//logr.Debugf("cluster health (%s): %s", method, res.Status)
}

// GetLogs returns a stream of log line records.
func (c *TrainingDataService) GetLogs(in *tds.Query, stream tds.TrainingData_GetLogsServer) (error) {
	start := time.Now()

	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService)).
		WithField(logger.LogkeyTrainingID, in.Meta.TrainingId).
		WithField(logger.LogkeyUserID, in.Meta.UserId)

	c.reportOnCluster("GetLogs", logr)

	//noinspection GoBoolExpressions
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

	res, err := c.executeQuery(indexName, query, isBackward, pos,
		pagesize, docTypeLog, shouldPostSortProcess, dlogr)

	doneQuery := time.Now()

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
			dlogr.Debugf("GetLogs: %d (%s): %s",
				logLineRecord.Meta.Rindex, logLineRecord.Meta.Subid, makeSnippetForDebug(logLineRecord.Line, 7))
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
	c.reportTime(logr, "GetLogs", in.Meta.TrainingId, in.Meta.Rindex, in.Meta.Time, start, doneQuery)

	dlogr.Debugf("function exit")
	return nil
}

// GetEMetrics returns a stream of evaluation metrics records.
func (c *TrainingDataService) GetEMetrics(in *tds.Query, stream tds.TrainingData_GetEMetricsServer) (error) {
	start := time.Now()
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService)).
		WithField(logger.LogkeyTrainingID, in.Meta.TrainingId).
		WithField(logger.LogkeyUserID, in.Meta.UserId)
	c.reportOnCluster("GetEMetrics", logr)
	//noinspection GoBoolExpressions
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

	res, err := c.executeQuery(indexName, query, isBackward, pos, pagesize,
		docTypeEmetrics, shouldPostSortProcess, dlogr)

	doneQuery := time.Now()

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
			dlogr.Debugf("Sending record with rindex %d, time %s",
				emetricsRecord.Meta.Rindex,
					emetricsRecord.Meta.Time)

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
	c.reportTime(logr, "GetEMetrics", in.Meta.TrainingId, in.Meta.Rindex, in.Meta.Time, start, doneQuery)

	dlogr.Debugf("function exit")
	return nil
}

func makeSnippetForDebug(str string, maxLen int) string {
	//noinspection GoBoolExpressions
	if TdsDebugMode {
		str = strings.TrimSpace(str)
		if str != "" {
			endIndex := maxLen
			if endIndex >= len(str) {
				endIndex = len(str) - 1
			}
			return str[:endIndex]
		}
	}
	return ""
}

// AddEMetrics adds the passed evaluation metrics record to storage.
func (c *TrainingDataService) AddEMetrics(ctx context.Context, in *tds.EMetrics) (*tds.AddResponse, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService)).
		WithField(logger.LogkeyTrainingID, in.Meta.TrainingId).
		WithField(logger.LogkeyUserID, in.Meta.UserId)
	c.reportOnCluster("AddEMetrics", logr)

	//noinspection GoBoolExpressions
	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)

	start := time.Now()

	jsonBytes, err := json.Marshal(in)
	//noinspection GoBoolExpressions
	if TdsDebugMode {
		dlogr.Debugf("emetrics_in_json: %d: %d: %s",
			in.Meta.Rindex, in.Meta.Time, makeSnippetForDebug(string(jsonBytes), 20))
	}

	_, err = c.es.Index().
		Index(indexName).
		Type(docTypeEmetrics).
		BodyString(string(jsonBytes)).
		Do(ctx)

	doneQuery := time.Now()

	out := new(tds.AddResponse)
	if err != nil {
		logr.WithError(err).Errorf("Failed to add log line to elasticsearch")
		out.Success = false
		return out, err
	}

	out.Success = true

	c.reportTime(logr, "AddEMetrics", in.Meta.TrainingId, in.Meta.Rindex, in.Meta.Time, start, doneQuery)

	return out, nil
}

// AddLogLine adds the line line record to storage.
func (c *TrainingDataService) AddLogLine(ctx context.Context, in *tds.LogLine) (*tds.AddResponse, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	c.reportOnCluster("AddLogLine", logr)

	//noinspection GoBoolExpressions
	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)
	dlogr.Debugf("AddLogLine: %d (%s): %s", in.Meta.Rindex, in.Meta.Subid, makeSnippetForDebug(in.Line, 7))

	start := time.Now()

	out := new(tds.AddResponse)

	_, err := c.es.Index().
		Index(indexName).
		Type(docTypeLog).
		BodyJson(in).
		Do(ctx)

	doneQuery := time.Now()

	if err != nil {
		logr.WithError(err).Errorf("Failed to add log line to elasticsearch")
		out.Success = false
		return out, err
	}

	out.Success = true

	c.reportTime(logr, "AddLogLine", in.Meta.TrainingId, in.Meta.Rindex, in.Meta.Time, start, doneQuery)
	dlogr.Debugf("exit")
	return out, nil
}

// AddEMetricsBatch adds the passed evaluation metrics record to storage.
//noinspection GoBoolExpressions
func (c *TrainingDataService) AddEMetricsBatch(ctx context.Context,
	inBatch *tds.EMetricsBatch) (*tds.AddResponse, error) {

	out := new(tds.AddResponse)
	if len(inBatch.Emetrics) == 0 {
		out.Success = true
		return out, nil
	}
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService)).
		WithField(logger.LogkeyTrainingID, inBatch.Emetrics[0].Meta.TrainingId).
		WithField(logger.LogkeyUserID, inBatch.Emetrics[0].Meta.UserId)
	c.reportOnCluster("AddEMetrics", logr)

	//noinspection GoBoolExpressions
	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)

	if TdsDebugEMetricAdd {
		fmt.Printf("------------------\n")
	}
	start := time.Now()
	//bulkRequest := c.es.Bulk()
	bulkRequest := c.es.Bulk().Index(indexName).Type(docTypeEmetrics)
	for _, in := range inBatch.Emetrics {

		jsonBytes, err := json.Marshal(in)
		if err != nil {
			logr.WithError(err).Errorf("Could not marshal request to string: %+v", *in)
			out.Success = false
			return out, err
		}
		if TdsDebugEMetricAdd {
			var jsonBytes []byte
			//noinspection GoBoolExpressions
			fmt.Printf("emetrics_in_json: %d: %d: %s\n",
				in.Meta.Rindex, in.Meta.Time, makeSnippetForDebug(string(jsonBytes), 20))
		}

		r := es.NewBulkIndexRequest().
			Index(indexName).
			Type(docTypeEmetrics).
			Doc(string(jsonBytes))

		bulkRequest = bulkRequest.Add(r)

	}
	bulkResponse, err := bulkRequest.Refresh("wait_for").Do(ctx)
	if err != nil {
		logr.WithError(err).Error("bulkRequest.Refresh returned error")
	}
	if bulkResponse == nil {
		logr.Warning("expected bulkResponse to be != nil; got nil")
	}
	c.refreshIndexes(ctx, dlogr)

	doneQuery := time.Now()
	if TdsDebugEMetricAdd {
		fmt.Printf("------------------\n")
	}

	//if inBatch.Force {
	//	c.esBulkProcessor.Flush()
	//}

	out.Success = true

	c.reportTime(logr, "AddEMetrics", inBatch.Emetrics[0].Meta.TrainingId,
		inBatch.Emetrics[0].Meta.Rindex, inBatch.Emetrics[0].Meta.Time, start, doneQuery)

	return out, nil
}

// AddLogLineBatch adds the line line record to storage.
//noinspection GoBoolExpressions
func (c *TrainingDataService) AddLogLineBatch(ctx context.Context,
	inBatch *tds.LogLineBatch) (*tds.AddResponse, error) {

	out := new(tds.AddResponse)
	if len(inBatch.LogLine) == 0 {
		out.Success = true
		return out, nil
	}
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	c.reportOnCluster("AddLogLine", logr)

	//noinspection GoBoolExpressions
	dlogr := makeDebugLogger(logr.Logger, TdsDebugMode)

	var err error
	if TdsDebugLogLineAdd {
		fmt.Printf("------------------\n")
	}
	start := time.Now()
	prevTimeStamp := int64(0)
	bulkRequest := c.es.Bulk().Index(indexName).Type(docTypeEmetrics)
	for _, in := range inBatch.LogLine {
		if TdsDebugLogLineAdd {
			if prevTimeStamp == 0 {
				prevTimeStamp = in.Meta.Time
			}

			timeSincePrev := in.Meta.Time - prevTimeStamp
			fmt.Printf("AddLogLineBatch: %d (tmSncPrev: %d)(%s): %s\n", in.Meta.Rindex, int(timeSincePrev),
				in.Meta.Subid, makeSnippetForDebug(in.Line, 7))

			prevTimeStamp = in.Meta.Time
		}
		jsonBytes, err := json.Marshal(in)
		if err != nil {
			logr.WithError(err).Errorf("Could not marshal request to string: %+v", *in)
			out.Success = false
			return out, err
		}

		r := es.NewBulkIndexRequest().
			Index(indexName).
			Type(docTypeLog).
			Doc(string(jsonBytes))

		bulkRequest.Add(r)
	}
	bulkResponse, err := bulkRequest.Refresh("wait_for").Do(ctx)
	if err != nil {
		logr.WithError(err).Error("bulkRequest.Refresh returned error")
	}
	if bulkResponse == nil {
		logr.Warning("expected bulkResponse to be != nil; got nil")
	}
	c.refreshIndexes(ctx, dlogr)

	doneQuery := time.Now()
	if TdsDebugLogLineAdd {
		fmt.Printf("------------------\n")
	}

	//if inBatch.Force {
	//	c.esBulkProcessor.Flush()
	//}

	out.Success = true

	c.reportTime(logr, "AddLogLine", inBatch.LogLine[0].Meta.TrainingId,
		inBatch.LogLine[0].Meta.Rindex, inBatch.LogLine[0].Meta.Time, start, doneQuery)

	return out, err
}


// DeleteEMetrics deletes the queried evaluation metrics from storage.
func (c *TrainingDataService) DeleteEMetrics(ctx context.Context, in *tds.Query) (*tds.DeleteResponse, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService)).
		WithField(logger.LogkeyTrainingID, in.Meta.TrainingId).
		WithField(logger.LogkeyUserID, in.Meta.UserId)
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
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService)).
		WithField(logger.LogkeyTrainingID, in.Meta.TrainingId).
		WithField(logger.LogkeyUserID, in.Meta.UserId)
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
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService)).
		WithField(logger.LogkeyTrainingID, in.Meta.TrainingId).
		WithField(logger.LogkeyUserID, in.Meta.UserId)
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

func createIndexWithLogsIfDoesNotExist(ctx context.Context, client *es.Client) error {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))
	logr.Debugf("function entry")
	
	mainIndex := indexNameV1

	logr.Infof("calling IndexExists for %s", mainIndex)

	exists, err := client.IndexExists(mainIndex).Do(ctx)
	if err != nil {
		logr.WithError(err).Errorf("IndexExists for %s failed", mainIndex)
	}

	if exists {
		// ignore error if already exist
		client.Alias().Add(mainIndex, indexName).Do(ctx)
		if err != nil {
			logr.WithError(err).Infof("alias alias for %s failed", mainIndex)
		}

		logr.Infof("Maintaining index: %s", mainIndex)
		 return nil
	}

	logr.Debugf("calling CreateIndex")
	ires, err := client.CreateIndex(mainIndex).Do(ctx)
	if err != nil {
		logr.WithError(err).Debug("CreateIndex failed")
		return err
	}
	if !ires.Acknowledged {
		return errors.New("the put mapping was not acknowledged")
	}
	res, err := client.PutMapping().Index(mainIndex).Type(docTypeLog).BodyString(indexMappingLogs).Do(ctx)
	if err != nil {
		logr.WithError(err).Debug("PutMapping logs failed")
		return err
	}
	if !res.Acknowledged {
		return errors.New("the put mapping was not acknowledged")
	}
	res, err = client.PutMapping().Index(mainIndex).Type(docTypeEmetrics).BodyString(indexMappingEmetrics).Do(ctx)
	if err != nil {
		logr.WithError(err).Debug("PutMapping logs failed")
		return err
	}
	if !res.Acknowledged {
		return errors.New("the put mapping was not acknowledged")
	}
	client.Alias().Remove(indexNameV1, indexName).Do(ctx)
	client.Alias().Add(mainIndex, indexName).Do(ctx)

	exists, err = client.IndexExists(indexName).Do(context.Background())
	if err != nil {
		logr.WithError(err).Errorf("IndexExists for %s failed", indexName)
	}
	logr.Infof("after creation of index %s, exists: %t", indexName, exists)

	if false {
		existsV1, err := client.IndexExists(indexNameV1).Do(ctx)
		//existsV1, err := client.IndexExists(indexNameV1).Do(context.Background())
		if err != nil {
			logr.WithError(err).Errorf("IndexExists for %s failed", indexNameV1)
		}
		if existsV1 {
			logr.Infof("reindex from %s to %s", indexNameV1, indexNameV2)

			src := es.NewReindexSource().Index(indexNameV1)
			dst := es.NewReindexDestination().Index(indexNameV2)
			res, err := client.Reindex().Source(src).Destination(dst).Refresh("true").Do(context.Background())

			if err != nil {
				logr.WithError(err).Debugf("Reindex from %s to %s failed", indexNameV1, indexNameV2)
			}
			logr.Infof("Reindex of %ld documents took %ld",
				res.Total, res.Took)

			logr.Infof("deleting index %s", indexNameV1)
			client.DeleteIndex(indexNameV1).Do(ctx)
		}
	}

	logr.Debugf("function exit")
	return err
}

