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

package logger

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/IBM/FfDL/commons/config"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Constants for logging
const (
	// ==============================
	// ======= Log field keys =======
	// These keys are to be used for structured logging, and will
	// be visible to the Elastic Search repository for query and the like.
	LogkeyCaller           = "caller_info"
	LogkeyTrainingID       = "training_id"
	LogkeyGpuType          = "gpu_type"
	LogkeyGpuUsage         = "gpu_usage"
	LogkeyFramework        = "framework_name"
	LogkeyFrameworkVersion = "framework_version"
	LogkeyErrorCode        = "error_code"
	LogkeyErrorType        = "error_type"
	LogkeyModelID          = "model_id"
	LogkeyModelName        = "model_name"
	LogkeyDBName           = "db_name"
	LogkeyUserID           = "user_id"
	LogkeyModule           = "module"
	LogkeyIsMetrics        = "is_metrics"
	LogkeyIsFollow         = "is_follow"
	LogkeyIsSummary        = "is_summary"
	LogkeyObjectstorePath  = "os_path"
	LogkeyTrainingDataURI  = "training_data_uri"
	LogkeyModelDataURI     = "model_data_uri"
	LogkeyModelFilename    = "model_filename"

	LogkeyDeployerService      = "deployer-service"
	LogkeyLcmService           = "lifecycle-manager-service"
	LogkeyRestAPIService       = "rest-api"
	LogkeyStorageService       = "storage-service"
	LogkeyTrainerService       = "trainer-service"
	LogkeyVolumeManagerService = "volume-manager-service"

	LogkeyJobMonitor = "jobmonitor"

	// ==============================
	// ======= Log categories =======

	// These keys are to be used to enable specific keyed log categories, and are meant to
	// be used as the second argument to `LocLoggerCategorized(logr *log.Entry, isEnabledKey string)`.
	// For example, if I'm trying to debug code associated with training logs, the dev can
	// set DLAAS_LOG_GETTRAININGLOGSTREAM to true in the environment when the LCM is launched.
	// (There may also be a way to set it dynamically, I'm not sure).  This capability is in addition
	// to the log levels, such as logrus.DebugLevel, logrus.WarnLevel, etc., which are still applicable.

	// DLAAS_LOG_GETTRAININGLOGSTREAM=true
	LogCategoryGetTrainingLogStream = "log_GetTrainingLogStream"
	LogCategoryGetTrainingLogStreamDefaultValue = false

	// DLAAS_LOG_GETTRAININGLOGSTREAMFROMOBJSTORE=true
	LogCategoryGetTrainingLogStreamFromObjStore = "log_getTrainingLogStreamFromObjStore"
	LogCategoryGetTrainingLogStreamFromObjStoreDefaultValue = false

	// DLAAS_LOG_REPOSITORY=true
	LogCategoryRepository = "log_repository"
	LogCategoryRepositoryDefaultValue = false

	// DLAAS_LOG_SERVELOGHANDLER=true
	LogCategoryServeLogHandler = "log_serveLogHandler"
	LogCategoryServeLogHandlerDefaultValue = false
)

// Ensure that it is initialized only once
var loggerInitOnce sync.Once

// FileInfoFindGood looks at the stack, and tries to find the first entry that
// is not infrastructure related.  i.e. that is essentially application code.
func FileInfoFindGood() string {
	// Inspect runtime call stack
	pc := make([]uintptr, 30)
	stackDepth := runtime.Callers(0, pc)

	// Try and skip functions on the stack that are clearly infrastructure related.  Typically the stack will
	// look something like:
	// stack[0]: /usr/local/go/src/runtime/extern.go:219
	// stack[1]: /home/sboag/git/src/github.com/IBM/FfDL/commons/logger/logger.go:72
	// stack[2]: /home/sboag/git/src/github.com/IBM/FfDL/commons/logger/logger.go:120
	// stack[3]: /home/sboag/git/src/github.com/IBM/FfDL/commons/logger/logger.go:136
	// stack[4]: /home/sboag/git/src/github.com/IBM/FfDL/commons/service/lcm/service_impl.go:656
	// stack[5]: /usr/local/go/src/runtime/asm_amd64.s:2087
	// We're only interested in #4 here.
	// Enable this if you want to view the stack
	//fmt.Print("==============\n")
	//for i := 0; i < stackDepth; i++ {
	//	f := runtime.FuncForPC(pc[i])
	//	file, line := f.FileLine(pc[i])
	//	fmt.Printf("stack[%d]: %s:%d\n", i, file, line)
	//}

	var file string
	var line int
	var f *runtime.Func
	for i := 0; i < stackDepth; i++ {
		f = runtime.FuncForPC(pc[i])
		file, line = f.FileLine(pc[i])
		// There might ought to be some sort of registry for these hard-coded patterns.
		if strings.Contains(file, ".pb.") {
			continue
		}
		if strings.Contains(file, "runtime/extern.go") {
			continue
		}
		if strings.Contains(file, "logger/logger.go") {
			continue
		}
		if strings.Contains(file, "/logging_impl.go") {
			continue
		}
		if strings.Contains(file, "/logging.go") {
			continue
		}
		break
	}

	// Truncate abs file path
	if slash := strings.LastIndex(file, "/"); slash >= 0 {
		file2 := file[slash:]
		clipped := file[:slash]
		// But, better if it's qualified one level, given some generic filenames in DLaaS.
		// Could go up to DLaaS root, I suppose, but would be pretty verbose.
		if slash := strings.LastIndex(clipped, "/"); slash >= 0 {
			file2 = clipped[slash+1:]+file2
		}
		file = file2
	}

	// Truncate package name
	funcName := f.Name()
	if slash := strings.LastIndex(funcName, "."); slash >= 0 {
		funcName = funcName[slash+1:]
	}

	return fmt.Sprintf("%s:%d %s -", file, line, funcName)
}

func LogStackTrace() {
	pc := make([]uintptr, 30)
	stackDepth := runtime.Callers(0, pc)
	for i := 0; i < stackDepth; i++ {
		f := runtime.FuncForPC(pc[i])
		file, line := f.FileLine(pc[i])
		// Truncate package name
		funcName := f.Name()
		if slash := strings.LastIndex(funcName, "."); slash >= 0 {
			funcName = funcName[slash+1:]
		}
		locString := fmt.Sprintf("%s:%d %s -", file, line, funcName)
		log.Debugf("   ---> %s", locString)
	}
}

// NewDlaaSLogData construct log data object.
func NewDlaaSLogData(serviceName string) log.Fields {
	f := FileInfoFindGood()
	data := log.Fields{LogkeyCaller: f}

	data[LogkeyModule] = serviceName
	return data
}

// LogServiceBasic Construct new basic logger for service.
func LogServiceBasic(serviceName string) *log.Entry {
	data := NewDlaaSLogData(serviceName)

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

// LogServiceBasicWithFields Construct new basic logger for service, with key/value pairs.
func LogServiceBasicWithFields(serviceName string, fields log.Fields) *log.Entry {
	data := NewDlaaSLogData(serviceName)

	for k, v := range fields {
		data[k] = v
	}
	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

// Config initializes the logging by setting the env var LOGLEVEL=<level>.
// Valid level values are: panic, fatal, error, warn, info, debug
func Config() {

	loggerInitOnce.Do(func() {

		level := viper.GetString(config.LogLevelKey)
		if level != "" {
			val, err := log.ParseLevel(level)
			if err != nil {
				panic(fmt.Sprintf("Error setting up logger: %s", err.Error()))
			}
			log.SetLevel(val)
		}

		loggingType := viper.GetString(config.LoggingType)
		switch loggingType {
		case config.LoggingTypeJson:
			log.SetFormatter(&log.JSONFormatter{})
		case config.LoggingTypeText:
			log.SetFormatter(&log.TextFormatter{})
		default: // any other env will use local settings (assuming outside SL)
			env := viper.GetString(config.EnvKey)

			switch env {
			case config.DevelopmentEnv:
				log.SetFormatter(&log.JSONFormatter{})
			case config.StagingEnv:
				log.SetFormatter(&log.JSONFormatter{})
			case config.ProductionEnv:
				log.SetFormatter(&log.JSONFormatter{})
			default: // any other env will use local settings (assuming outside SL)
				log.SetFormatter(&log.JSONFormatter{})
			}
		}

		viper.SetDefault(LogCategoryGetTrainingLogStream, LogCategoryGetTrainingLogStreamDefaultValue)
		viper.SetDefault(LogCategoryGetTrainingLogStreamFromObjStore,
			LogCategoryGetTrainingLogStreamFromObjStoreDefaultValue)
		viper.SetDefault(LogCategoryRepository, LogCategoryRepositoryDefaultValue)

		viper.SetDefault(LogCategoryServeLogHandler, LogCategoryServeLogHandlerDefaultValue)
	})
	// otherwise use logrus' default value
}

// LocLoggingEntry wraps another logger, and add code location information about where the the log entry occurs.
// Because logging can be expensive, especially logging that needs to figure out
// it's location, this class also has a way to disable the logging for debug/informational
// entries.
type LocLoggingEntry struct {
	Logger *log.Entry

	// If true, enable functions at the level of debug, info, and print.
	// Warn, Error, Fatal, and Panic are always enabled.
	Enabled bool
}

// LocLogger simply creates a LocLoggingEntry that wraps another logger.
func LocLogger(logr *log.Entry) *LocLoggingEntry {
	logger := new(LocLoggingEntry)
	logger.Logger = logr
	logger.Enabled = true

	return logger
}

// LocLoggerCategorized creates a LocLoggingEntry that wraps another logger, and accepts
// a string that is used with viper to say whether the informational/debug logging
// is enabled for this logger.
func LocLoggerCategorized(logr *log.Entry, isEnabledKey string) *LocLoggingEntry {
	logger := new(LocLoggingEntry)
	logger.Logger = logr
	logger.Enabled = viper.GetBool(isEnabledKey)

	return logger
}

// MakeNew makes a new LocLoggingEntry from an existing LocLoggingEntry,
// but using a new inner logger.
func (entry *LocLoggingEntry) MakeNew(logr *log.Entry) *LocLoggingEntry {
	logger := new(LocLoggingEntry)
	logger.Logger = logr
	logger.Enabled = entry.Enabled

	return logger
}

// withLoc returns a new logger that has recorded the code location where it occurs.
func (entry *LocLoggingEntry) withLoc() *LocLoggingEntry {
	f := FileInfoFindGood()
	data := log.Fields{LogkeyCaller: f}

	newLogger := entry.Logger.WithFields(data)
	return entry.MakeNew(newLogger)
}

// Add an error as single field (using the key defined in ErrorKey) to the Entry.
func (entry *LocLoggingEntry) WithError(err error) *LocLoggingEntry {
	return entry.MakeNew(entry.Logger.WithError(err))
}

// Add a single field to the Entry.
func (entry *LocLoggingEntry) WithField(key string, value interface{}) *LocLoggingEntry {
	return entry.MakeNew(entry.Logger.WithField(key, value))
}

// Add a map of fields to the Entry.
func (entry *LocLoggingEntry) WithFields(fields log.Fields) *LocLoggingEntry {
	return entry.MakeNew(entry.Logger.WithFields(fields))
}

func (entry *LocLoggingEntry) Debug(args ...interface{}) {
	if entry.Enabled == true {
		entry.withLoc().Logger.Debug(args...)
	}
}

func (entry *LocLoggingEntry) Print(args ...interface{}) {
	if entry.Enabled == true {
		entry.withLoc().Logger.Print(args...)
	}
}

func (entry *LocLoggingEntry) Info(args ...interface{}) {
	if entry.Enabled == true {
		entry.withLoc().Logger.Info(args...)
	}
}

func (entry *LocLoggingEntry) Warn(args ...interface{}) {
	entry.withLoc().Logger.Warn(args...)
}

func (entry *LocLoggingEntry) Warning(args ...interface{}) {
	entry.withLoc().Logger.Warning(args...)
}

func (entry *LocLoggingEntry) Error(args ...interface{}) {
	entry.withLoc().Logger.Error(args...)
}

func (entry *LocLoggingEntry) Fatal(args ...interface{}) {
	entry.withLoc().Logger.Fatal(args...)
}

func (entry *LocLoggingEntry) Panic(args ...interface{}) {
	entry.withLoc().Logger.Panic(args...)
}

// Entry Printf family functions

func (entry *LocLoggingEntry) Debugf(format string, args ...interface{}) {
	if entry.Enabled == true {
		entry.withLoc().Logger.Debugf(format, args...)
	}
}

func (entry *LocLoggingEntry) Infof(format string, args ...interface{}) {
	if entry.Enabled == true {
		entry.withLoc().Logger.Infof(format, args...)
	}
}

func (entry *LocLoggingEntry) Printf(format string, args ...interface{}) {
	if entry.Enabled == true {
		entry.withLoc().Logger.Printf(format, args...)
	}
}

func (entry *LocLoggingEntry) Warnf(format string, args ...interface{}) {
	entry.withLoc().Logger.Warnf(format, args...)
}

func (entry *LocLoggingEntry) Warningf(format string, args ...interface{}) {
	entry.withLoc().Logger.Warningf(format, args...)
}

func (entry *LocLoggingEntry) Errorf(format string, args ...interface{}) {
	entry.withLoc().Logger.Errorf(format, args...)
}

func (entry *LocLoggingEntry) Fatalf(format string, args ...interface{}) {
	entry.withLoc().Logger.Fatalf(format, args...)
}

func (entry *LocLoggingEntry) Panicf(format string, args ...interface{}) {
	entry.withLoc().Logger.Panicf(format, args...)
}

// Entry Println family functions

func (entry *LocLoggingEntry) Debugln(args ...interface{}) {
	if entry.Enabled == true {
		entry.withLoc().Logger.Debugln(args...)
	}
}

func (entry *LocLoggingEntry) Infoln(args ...interface{}) {
	if entry.Enabled == true {
		entry.withLoc().Logger.Infoln(args...)
	}
}

func (entry *LocLoggingEntry) Println(args ...interface{}) {
	if entry.Enabled == true {
		entry.withLoc().Logger.Println(args...)
	}
}

func (entry *LocLoggingEntry) Warnln(args ...interface{}) {
	entry.withLoc().Logger.Warnln(args...)
}

func (entry *LocLoggingEntry) Warningln(args ...interface{}) {
	entry.withLoc().Logger.Warnln(args...)
}

func (entry *LocLoggingEntry) Errorln(args ...interface{}) {
	entry.withLoc().Logger.Errorln(args...)
}

func (entry *LocLoggingEntry) Fatalln(args ...interface{}) {
	entry.withLoc().Logger.Fatalln(args...)
}

func (entry *LocLoggingEntry) Panicln(args ...interface{}) {
	entry.withLoc().Logger.Panicln(args...)
}
