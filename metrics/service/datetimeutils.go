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
	"fmt"
	"strconv"
	"regexp"
	"strings"
	"time"
	"github.com/IBM/FfDL/commons/logger"
)

func makeMillisecondTime(timeObj time.Time) int64 {
	// return timeObj.UnixNano() / int64(time.Millisecond)
	return timeObj.Unix()
}

// Make attempt to extract milliseconds from "since' string
func humanStringToUnixTime(humanString string) (int64, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(LogkeyTrainingDataService))

	var unixTime int64
	var err error
	var durationObj time.Duration
	var timeObj time.Time

	humanString = strings.TrimSpace(humanString)

	// nFound, err := fmt.Sscanf(humanString,"%d", &unixTime)
	unixMillisecondRegEx, err := regexp.Compile("^(\\d+)$")
	if err != nil {
		logr.WithError(err).Printf("Error in regex\n")
	} else {
		foundStrings := unixMillisecondRegEx.FindStringSubmatch(humanString)
		if foundStrings != nil {
			quantityStr := foundStrings[0]
			unixTime, _ = strconv.ParseInt(quantityStr, 10, 64)
		}
	}

	if unixTime == 0 {
		//ANSIC       = "Mon Jan _2 15:04:05 2006"
		//UnixDate    = "Mon Jan _2 15:04:05 MST 2006"
		//RubyDate    = "Mon Jan 02 15:04:05 -0700 2006"
		//RFC822      = "02 Jan 06 15:04 MST"
		//Kitchen     = "3:04PM"
		//// Handy time stamps.
		//Stamp      = "Jan _2 15:04:05"
		//StampMilli = "Jan _2 15:04:05.000"
		//StampMicro = "Jan _2 15:04:05.000000"
		//StampNano  = "Jan _2 15:04:05.000000000"
		var tformats = [...]string {
			time.ANSIC,
			time.UnixDate,
			time.RubyDate,
			time.RFC822,
			time.Kitchen,
			//time.Stamp,
			//time.StampMilli,
			//time.StampMicro,
			//time.StampNano,
		}

		for _, tformat := range tformats {
			humanStringLower := strings.ToUpper(humanString)
			timeObj, err = time.ParseInLocation(tformat, humanStringLower, time.Local)
			if err != nil {
				logr.WithError(err).Printf("No: %s\n", tformat)
			} else {
				logr.Debugf("Yes: %s: %v\n", humanString, timeObj)
				unixTime = makeMillisecondTime(timeObj)
				break
			}
		}
	}

	if unixTime == 0 {
		durationObj, err = time.ParseDuration(humanString)
		if err != nil {
			logr.WithError(err).Printf("No\n")
		} else {
			logr.Debugf("%s: %v", humanString, durationObj)
			unixTime = makeMillisecondTime(time.Now().Add(-durationObj))
			err = nil
		}
	}

	if unixTime == 0 {
		if strings.ToLower(humanString) == "now" {
			unixTime = makeMillisecondTime(time.Now())
			err = nil
		}
	}

	if unixTime == 0 {
		if strings.ToLower(humanString) == "yesterday" {
			unixTime = makeMillisecondTime(time.Now().Add(-(time.Hour * 24)))
			err = nil
		}
	}

	if unixTime == 0 {
		logr.Debugf("Scanning using regex")
		regExStr := "^(\\d+)\\s+(\\w+)\\s+ago$"
		durRegEx, err := regexp.Compile(regExStr)
		// durationObj, err = time.ParseDuration(humanString)
		if err != nil {
			logr.WithError(err).Printf("Error in regex: %s\n", regExStr)
		} else {
			humanString = strings.ToLower(humanString)
			foundStrings := durRegEx.FindStringSubmatch(humanString)
			if foundStrings != nil {
				quantityStr := foundStrings[1]
				quantity, err := strconv.ParseInt(quantityStr, 10, 64)
				isErr := false
				if err != nil {
					logr.WithError(err).Printf("Not able to parse amount: %s\n", quantityStr)
					isErr = true
				} else {
					unitTypeStr := foundStrings[2]
					unitTypeStr = strings.ToLower(unitTypeStr)
					switch unitTypeStr {
					case "s":
					case "sec":
					case "secs":
					case "second":
					case "seconds":
						humanString = fmt.Sprintf("%vs", quantity)
						break
					case "m":
					case "min":
					case "mins":
					case "minute":
					case "minutes":
						humanString = fmt.Sprintf("%vm", quantity)
						break
					case "h":
					case "hour":
					case "hours":
						humanString = fmt.Sprintf("%vm", quantity*60)
						break
					case "d":
					case "day":
					case "days":
						humanString = fmt.Sprintf("%vm", quantity*24*60)
						break
					case "week":
					case "weeks":
						humanString = fmt.Sprintf("%vm", quantity*7*24*60)
						break
					default:
						logr.WithError(err).Printf("Unit type unknown\n")
						isErr = true
					}
				}
				if !isErr {
					logr.Debugf("Parsed ago string: %s\n", humanString)
					durationObj, err = time.ParseDuration(humanString)
					if err != nil {
						logr.WithError(err).Printf("No\n")
					} else {
						logr.Debugf("%s: %v", humanString, durationObj)
						unixTime = makeMillisecondTime(time.Now().Add(-durationObj))
						err = nil
					}
				}
			} else {
				logr.Debugf("Not able to parse with regex: %s\n", regExStr)
			}
		}
	}

	if unixTime != 0 {
		backToTimeObj := time.Unix(0, unixTime*int64(time.Millisecond))
		durationSince := time.Since(backToTimeObj)
		logr.Debugf("reverse duration:%v", durationSince)
	}

	if unixTime != 0 {
		err = nil
	}

	return unixTime, err
}
