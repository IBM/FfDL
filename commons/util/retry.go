package util

import (
	"fmt"
	"time"

	"github.com/IBM/FfDL/commons/logger"
)

//
// Helper retry function
//
func Retry(attempts int, interval time.Duration, description string, logr *logger.LocLoggingEntry, callback func() error) (err error) {
	for i := 0; ; i++ {
		err = callback()
		if err == nil {
			return nil
		}
		if i >= (attempts - 1) {
			break
		}
		time.Sleep(interval)
		logr.Warnf("Retrying function %s due to error %s", description, err)
	}
	return fmt.Errorf("function %s after %d attempts, last error: %s", description, attempts, err)
}
