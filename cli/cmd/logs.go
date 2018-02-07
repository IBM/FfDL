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

package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"time"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/bluemix/terminal"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/plugin"
	"fmt"
	"os"
	"net/url"
	"strings"
	"github.ibm.com/ffdl/ffdl-core/restapi/api_v1/client/models"
	"github.ibm.com/ffdl/ffdl-core/restapi/api_v1/restmodels"
	"net/http"
	dlaasClient "github.ibm.com/ffdl/ffdl-core/restapi/api_v1/client"
	"io"
	"encoding/base64"
	"github.com/go-openapi/runtime/client"

	"errors"
	"golang.org/x/net/websocket"
	"encoding/json"
	// "bytes"
	"bytes"
)

const (
	origin = "http://localhost/"
)

// LogsCmd represents the instance of this command
type LogsCmd struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewLogsCmd creates a new instance of this command
func NewLogsCmd(ui terminal.UI, context plugin.PluginContext) *LogsCmd {
	return &LogsCmd{
		ui:      ui,
		context: context,
	}
}

// Run is the handler for the training-show CLI command.
func (cmd *LogsCmd) Run(cliContext *cli.Context) error {
	cmd.config = cmd.context.PluginConfig()

	args := cliContext.Args()

	if len(args) == 0 {
		cmd.ui.Failed("Incorrect arguments")
	} else {
		log.Debugf("debug training-logs: flags: %v", cliContext.FlagNames())
		doWebSockets := cliContext.IsSet("follow")
		isMetrics := cliContext.IsSet("metrics")
		isJSON := cliContext.IsSet("json")
		log.Debugf("debug training-logs: isMetrics: %v", isMetrics)
		trainingID := args[0]
		log.Debugf("debug training-logs: (1) %s %t", trainingID, doWebSockets)

		if !isJSON {
			if isMetrics {
				cmd.ui.Say("Getting evaluation metrics for '%s'...", terminal.EntityNameColor(trainingID))
			} else {
				cmd.ui.Say("Getting model training logs for '%s'...", terminal.EntityNameColor(trainingID))
			}
		}
		dc, err := NewDlaaSClient()
		if err != nil {
			cmd.ui.Failed(err.Error())
		}
		if(trainingID != "wstest"){
			params := models.NewGetModelParams().WithModelID(trainingID).WithTimeout(defaultOpTimeout)
			_, err := dc.Models.GetModel(params, BasicAuth())

			if err != nil {
				log.WithError(err).Debug("dc.Models.GetModel(...) returned an error!")
				var s string
				switch err.(type) {
				case *models.GetModelUnauthorized:
					s = badUsernameOrPWD
				case *models.GetModelNotFound:
					s = "Model not found."
				}
				responseError(s, err, cmd.ui)
				return nil
			}
		}

		if doWebSockets {
			log.Debugf("debug training-logs: doing web socket")
			dlaasURL := os.Getenv("DLAAS_URL")
			if dlaasURL == "" {
				dlaasURL = dlaasClient.DefaultHost + dlaasClient.DefaultBasePath
			}
			if !strings.HasSuffix(dlaasURL, dlaasClient.DefaultBasePath) {
				dlaasURL = dlaasURL + dlaasClient.DefaultBasePath
			}
			u, _ := url.Parse(dlaasURL)
			host := u.Host
			log.Debugf("debug training-logs: host: %s", host)

			// what's the best way to do this?
			var streamHost string
			var wsprotocol string
			if strings.HasPrefix(host, "gateway.") || strings.HasPrefix(host, "gateway-") {
				streamHost = strings.Replace(host, "gateway", "stream", 1) + ":443"
			} else {
				streamHost = host
			}
			if strings.HasPrefix(streamHost, "stream.") || strings.HasPrefix(streamHost, "stream-") {
				wsprotocol = "wss"
			} else {
				wsprotocol = "ws"
			}

			var authStr string
			if u.User != nil {
				password, _ := u.User.Password()
				basicAuth = client.BasicAuth(u.User.Username(), password)
				authStr = base64.StdEncoding.EncodeToString([]byte(u.User.Username() + ":" + password))
			} else {
				username := os.Getenv("DLAAS_USERNAME")
				password := os.Getenv("DLAAS_PASSWORD")
				if username == "" || password == "" {
					return errors.New("Username and password not set")
				}
				basicAuth = client.BasicAuth(username, password)
				authStr = base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			}

			var logsOrMetrics string
			if isMetrics {
				logsOrMetrics = "metrics"
			} else {
				logsOrMetrics = "logs"
			}

			var fullpath = fmt.Sprintf("%sv1/models/%s/%s", u.Path, trainingID, logsOrMetrics)
			trainingLogURL := url.URL{Scheme: wsprotocol, Host: streamHost, Path: fullpath, RawQuery: "version=2017-03-17"}

			headers := make(http.Header, 2)
			headers.Set(watsonUserInfoHeader, watsonUserInfo)
			headers.Set("Authorization", "Basic "+authStr)

			log.Debugf("call: websocket.Dial: %s\n", trainingLogURL.String())

			c, err:= websocket.NewConfig(trainingLogURL.String(), origin)
			c.Header = headers
			c.Protocol = []string{ wsprotocol }
			if err != nil {
				return fmt.Errorf("Cannot configure websocket: %s", err)

			}
			ws, err := websocket.DialConfig(c)
			if err != nil {
				return fmt.Errorf("Cannot connect to remote websocket API: %s", err)
			}
			defer ws.Close()

			var msg = make([]byte, 1024*8)
			for {
				nread, err := ws.Read(msg)
				if err != nil {
					if err == io.EOF || strings.Contains(err.Error(), "close 1000 (normal)") {
						break
					}
					return fmt.Errorf("Reading training logs failed: %s", err.Error())
				}
				if isMetrics {
					if isJSON {
						var out bytes.Buffer
						json.Indent(&out, msg[:nread], "", "\t")
						out.WriteTo(os.Stdout)
					} else {
						var metricsRecords restmodels.MetricData

						err := json.Unmarshal(msg[:nread], &metricsRecords)
						if err != nil {
							fmt.Println("error:", err)
							fmt.Printf("Bad json: %s", msg[:nread])
							continue
						}

						if metricsRecords.Type == "Status" {
							fmt.Printf("Status: %s\n", metricsRecords.Values["Message"])
						} else {
							// fmt.Printf("dump: %+v\n", metricsRecords)
							fmt.Printf("Timestamp: %-12v "+
								"Type: %-8v "+
								"Iteration: %-7v",
								metricsRecords.Timestamp,
								metricsRecords.Type,
								metricsRecords.Iteration)

							// should probably sort these
							for k, v := range metricsRecords.Values {
								fmt.Printf("%s: %-12v", k, v)
							}
							fmt.Printf("\n")
						}

						// fmt.Printf("%+v\n", metricsRecords)
					}

				} else {
					fmt.Printf("%s", msg[:nread])
				}
			}
			fmt.Println()

		} else {
			if isMetrics {
				params :=
					models.NewGetMetricsParams().WithModelID(trainingID).WithTimeout(10 * time.Hour)
				log.Debugf("debug training-logs: doing http")
				r, w := io.Pipe()

				go func() {
					defer w.Close()
					_, err := dc.Models.GetMetrics(params, w)
					if err != nil {
						cmd.ui.Failed("Reading training logs failed(1): %s", err.Error())
					}
				}()

				time.Sleep(3 * time.Second)
				_, err = io.Copy(os.Stdout, r)
				if err != nil {
					cmd.ui.Failed("Reading training logs failed(2): %s", err.Error())
				}

			} else {
				params := models.NewGetLogsParams().WithModelID(trainingID).WithTimeout(10 * time.Hour)
				log.Debugf("debug training-logs: doing http")
				r, w := io.Pipe()

				go func(pw *io.PipeWriter) {
					defer pw.Close()
					_, err := dc.Models.GetLogs(params, pw)
					if err != nil {
						cmd.ui.Failed("Reading training logs failed(3): %s", err.Error())
					}
				}(w)

				time.Sleep(3 * time.Second)
				_, err = io.Copy(os.Stdout, r)
				if err != nil {
					cmd.ui.Failed("Reading training logs failed(4): %s", err.Error())
				}
			}
		}
		//cmd.ui.Ok() // don't print OK after as we do with other commands, as it may be interpreted as part
		// of the logs
	}
	return nil
}
