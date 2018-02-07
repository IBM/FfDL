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
 
 package middleware

import (
	"net/http"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	// UserIDHeader is the name of the HTTP header used to identify the user
	UserIDHeader         = "X-DLaaS-UserID"
	watsonUserInfoHeader = "X-Watson-Userinfo"
	watsonUserInfoParam  = "watson-userinfo"
	wmlUserIDHeader      = "X-WML-TenantID"
)

// AuthOptions for the auth middleware.
type AuthOptions struct {
	ExcludedURLs []string
}

// NewAuthMiddleware creates a new http.Handler that adds authentication logic to a given Handler
func NewAuthMiddleware(opts *AuthOptions) func(h http.Handler) http.Handler {
	if opts == nil {
		opts = &AuthOptions{ExcludedURLs: []string{}}
	}

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			log.Debugf("Enter into auth handler")
			log.Debugf("request: %+v", r)

			// check if the request URI matched excluded request URIs
			sort.Strings(opts.ExcludedURLs)
			i := sort.SearchStrings(opts.ExcludedURLs, r.RequestURI)
			if i < len(opts.ExcludedURLs) && opts.ExcludedURLs[i] == r.RequestURI {
				h.ServeHTTP(w, r)
				return // we are done
			}

			if r.Method == "OPTIONS" {
				if r.Header.Get("Access-Control-Request-Method") != "" {
					// TODO: Needs product stake-holder review

					log.Debugf("cors preflight detected")

					// cors preflight request/response
					w.Header().Add("Access-Control-Allow-Origin", "*")
					w.Header().Add("Access-Control-Allow-Methods", "PUT, GET, POST, DELETE, OPTIONS")
					w.Header().Add("Access-Control-Allow-Headers", "origin, x-requested-with, content-type, authorization, x-watson-userinfo, x-watson-authorization-token")
					w.Header().Add("Access-Control-Max-Age", "86400")

					w.Header().Add("Content-Type", "text/html; charset=utf-8")

					w.WriteHeader(200)

					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}

					return
				}
			}

			log.Debugf("Writing to header in callBefore \"Access-Control-Allow-Origin: *\"")

			w.Header().Add("Access-Control-Allow-Origin", "*")

			// Anything special to be done here for "x-watson-authorization-token"?

			// parse X-Watson-Userinfo header or watson-userinfo url param
			md := make(map[string]string)
			userInfoStr := r.Header.Get(watsonUserInfoHeader)
			if userInfoStr == "" {
				userInfoStr = r.FormValue(watsonUserInfoParam)
			}
			if userInfoStr != "" {
				userInfoPairs := strings.Split(userInfoStr, ";")
				for _, e := range userInfoPairs {
					pair := strings.Split(e, "=")
					if len(pair) == 2 {
						md[pair[0]] = pair[1]
					}
				}
			}

			// read required header and fail if not present
			userID := md["bluemix-instance-id"]
			if userID == "" {
				w.WriteHeader(401)
				w.Header().Add("Content-Type", "application/json; charset=utf-8")
				w.Write([]byte("{ \"message\" : \"Missing or malformed X-Watson-Userinfo header.\"}"))
				return
			}

			// if WML tenant ID is present use that as userID instead of bluemix-instance-id. This allows us
			// to integrate with WML in the interim and still being able to get through datapower.
			wmlTenantID := r.Header.Get(wmlUserIDHeader)
			log.Debugf("wmlTenantID: %s", wmlTenantID)
			if wmlTenantID != "" {
				log.Debugf("Found WML tenant ID '%s'. Using it as %s value", wmlTenantID, UserIDHeader)
				userID = wmlTenantID
			}

			// set the bluemix-instance-id as X-DlaaS header
			r.Header.Set(UserIDHeader, userID)
			log.Debugf("%s: %v", UserIDHeader, userID)

			if log.GetLevel() == log.DebugLevel {
				entry := log.NewEntry(log.StandardLogger())
				for k, v := range r.Header {
					entry = entry.WithField(k, v)
				}
				entry.Debug("Request headers:")
			}

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
