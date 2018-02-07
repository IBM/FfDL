package service

import (
	"fmt"
	"github.com/spf13/viper"
	"github.ibm.com/ffdl/ffdl-core/commons/config"
	"github.ibm.com/ffdl/ffdl-core/commons/logger"
	"github.ibm.com/ffdl/ffdl-core/metrics/client"
	"github.ibm.com/ffdl/ffdl-core/metrics/service/grpc_training_data_v1"
	"testing"
	"context"
	"time"
	"github.com/stretchr/testify/assert"
	log "github.com/sirupsen/logrus"
	//"io"
	// "encoding/json"
)

func init() {
	fmt.Printf("In test init\n")
	if false {
		viper.Set(config.DNSServerKey, "disabled")
	}
	logger.Config()
	log.SetLevel(log.DebugLevel)
	logr := logger.LocLogger(log.StandardLogger().WithField(logger.LogkeyModelName, "training-data-service-unit-test"))
	logr.Debugf("Initialization complete")
	fmt.Printf("Exit test init\n")
}


func initSomeData(ctx context.Context, c client.TrainingDataClient) {
	fakeTrainerID1 := "training-abcd"
	nMillisecondsPause := time.Millisecond*2
	for i := 0; i < 10; i++ {
		logRecord := grpc_training_data_v1.LogLine{
			Meta: &grpc_training_data_v1.MetaInfo{
				TrainingId: fakeTrainerID1,
				UserId:     "1234",
				Time:       time.Now().UnixNano() / int64(time.Millisecond),
				Rindex:    int64(i),
			},
			Line: fmt.Sprintf("A number: %d", i),
		}

		c.Client().AddLogLine(ctx, &logRecord)
		time.Sleep(nMillisecondsPause)
	}
	for i := 0; i < 10; i++ {
		emetricsRecord := grpc_training_data_v1.EMetrics{
			Meta: &grpc_training_data_v1.MetaInfo{
				TrainingId: fakeTrainerID1,
				UserId:     "1234",
				Time:       time.Now().UnixNano() / int64(time.Millisecond),
				Rindex:    int64(i),
			},
			Etimes:map[string]*grpc_training_data_v1.Any{
				"iteration": {Type:grpc_training_data_v1.Any_INT, Value: fmt.Sprintf("%v", i*10)},
				"blahblah": {Value: "hiya"},
				"anumber": {Type:grpc_training_data_v1.Any_FLOAT, Value: fmt.Sprintf("%v", 27.43)},
			},
			Grouplabel:"train",
			Values: map[string]*grpc_training_data_v1.Any{
				"accuracy": {Type:grpc_training_data_v1.Any_FLOAT, Value: fmt.Sprintf("%v", 0.999)},
				"areaUnderRoc": {Type:grpc_training_data_v1.Any_FLOAT, Value: fmt.Sprintf("%v", 0.129)},
			},
		}
		// fmt.Printf("record: %+v\n", emetricsRecord)

		c.Client().AddEMetrics(ctx, &emetricsRecord)
		time.Sleep(nMillisecondsPause)
	}


	fakeTrainerID1 = "training-tata"
	for i := 0; i < 10; i++ {
		logRecord := grpc_training_data_v1.LogLine{
			Meta: &grpc_training_data_v1.MetaInfo{
				TrainingId: fakeTrainerID1,
				UserId:     "1234",
				Time:       time.Now().UnixNano() / int64(time.Millisecond),
				Rindex:    int64(i),
			},
			Line: fmt.Sprintf("A number: %d", i),
		}

		c.Client().AddLogLine(ctx, &logRecord)
		time.Sleep(nMillisecondsPause)
	}
	for i := 0; i < 10; i++ {
		emetricsRecord := grpc_training_data_v1.EMetrics{
			Meta: &grpc_training_data_v1.MetaInfo{
				TrainingId: fakeTrainerID1,
				UserId:     "1234",
				Time:       time.Now().UnixNano() / int64(time.Millisecond),
				Rindex:    int64(i),
			},
			Etimes:map[string]*grpc_training_data_v1.Any{
				"iteration": {Type:grpc_training_data_v1.Any_INT, Value: fmt.Sprintf("%d", i*10)},
			},
			Grouplabel:"train",
			Values: map[string]*grpc_training_data_v1.Any{
				"accuracy": {Type:grpc_training_data_v1.Any_FLOAT, Value: fmt.Sprintf("%f", 0.999)},
				"areaUnderRoc": {Type:grpc_training_data_v1.Any_FLOAT, Value: fmt.Sprintf("%f", 0.129)},
			},
		}

		c.Client().AddEMetrics(ctx, &emetricsRecord)
		time.Sleep(nMillisecondsPause)
	}
	// Let the DB catch up
	time.Sleep(time.Second*1)

}

func deleteAllTrainingData(ctx context.Context, t *testing.T, c client.TrainingDataClient) {
	query := grpc_training_data_v1.Query {SearchType:grpc_training_data_v1.Query_ALL}
	var err error
	_, err = c.Client().DeleteJob(ctx, &query)
	assert.NoError(t, err)
}

func TestMetrics(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	logr := logger.LocLogger(log.StandardLogger().WithField(logger.LogkeyModelName, "training-data-service-unit-test"))
	logr.Debugf("Function entry")

	//ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	//defer cancel()
	//
	//c, err := client.NewTrainingDataClient()
	//assert.NoError(t, err)
	//defer c.Close()
	//
	//// deleteAllTrainingData(ctx, t, c)
	//
	//// initSomeData(ctx, c)
	//
	//query := grpc_training_data_v1.Query {
	//	Meta: &grpc_training_data_v1.MetaInfo {
	//		TrainingId: "training-4jip9Ybkg",
	//	},
	//}
	//
	//logr.Debugf("Calling c.Client().Get(ctx, &query)")
	//inStream, err := c.Client().GetEMetrics(ctx, &query)
	//assert.NoError(t, err)
	//
	//fmt.Printf("\n==================\n")
	//for {
	//	metrics, err := inStream.Recv()
	//	if err == io.EOF {
	//		break
	//	}
	//	if err != nil {
	//		logr.WithError(err).Errorf("Cannot read trained model log.")
	//		assert.NoError(t, err)
	//		break
	//	}
	//	fmt.Printf("training-id: %s\n", metrics.Meta.TrainingId)
	//	fmt.Printf("time: %d\n", metrics.Meta.Time)
	//	fmt.Printf("group-label: %s\n", metrics.Grouplabel)
	//
	//	var etimes map[string]*grpc_training_data_v1.Any
	//	etimes = metrics.Etimes
	//
	//	for k, v := range etimes {
	//		fmt.Printf("etime: %s: %s\n", k, v)
	//	}
	//
	//	var values map[string]*grpc_training_data_v1.Any
	//	values = metrics.Values
	//
	//	for k, v := range values {
	//		fmt.Printf("value: %s: %s\n", k, v)
	//	}
	//	//fmt.Printf("Expanded with +v: %+v\n", metrics)
	//	//
	//	//marshaledBytes, err := json.Marshal(metrics)
	//	//if err != nil {
	//	//	logr.WithError(err).Errorf("Metrics record can not be marshaled!")
	//	//	assert.NoError(t, err)
	//	//	break
	//	//}
	//	//fmt.Printf("Expanded with json.Marshal: %s\n", string(marshaledBytes))
	//
	//	fmt.Printf("-------\n")
	//
	//
	//	// fmt.Printf("%s, %d, %s\n", metrics.Meta.TrainingId, metrics.Meta.Time, metrics.Values["message"].Value)
	//}


	logr.Debugf("Function exit")
}
