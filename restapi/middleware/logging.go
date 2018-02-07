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
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

type timer interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

type realClock struct{}

func (rc *realClock) Now() time.Time {
	return time.Now()
}

func (rc *realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Middleware is a middleware handler that logs the request as it goes in and the response as it goes out.
type Middleware struct {
	Name string
	// Name is the name of the application as recorded in latency metrics
	Before func(*log.Entry, *http.Request, string) *log.Entry
	After  func(*log.Entry, ResponseWriter, time.Duration, string) *log.Entry

	logStarting bool

	clock timer

	// Exclude URLs from logging
	excludeURLs []string
}

// NewLoggingMiddleware returns a new *Middleware.
func NewLoggingMiddleware(name string) *Middleware {
	return &Middleware{
		Name:   name,
		Before: DefaultBefore,
		After:  DefaultAfter,

		logStarting: true,
		clock:       &realClock{},
	}
}

// SetLogStarting accepts a bool to control the logging of "started handling
// request" prior to passing to the next middleware
func (m *Middleware) SetLogStarting(v bool) {
	m.logStarting = v
}

// ExcludeURL adds a new URL u to be ignored during logging. The URL u is parsed, hence the returned error
func (m *Middleware) ExcludeURL(u string) error {
	if _, err := url.Parse(u); err != nil {
		return err
	}
	m.excludeURLs = append(m.excludeURLs, u)
	return nil
}

// ExcludedURLs returns the list of excluded URLs for this middleware
func (m *Middleware) ExcludedURLs() []string {
	return m.excludeURLs
}

// Handle is the main entrypoint for the middleware to process the request
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.Before == nil {
			m.Before = DefaultBefore
		}

		if m.After == nil {
			m.After = DefaultAfter
		}

		for _, u := range m.excludeURLs {
			if r.URL.Path == u {
				return
			}
		}

		start := m.clock.Now()

		// Try to get the real IP
		remoteAddr := r.RemoteAddr
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			remoteAddr = realIP
		}

		entry := log.NewEntry(log.StandardLogger())

		if reqID := r.Header.Get("X-Request-Id"); reqID != "" {
			entry = entry.WithField("request_id", reqID)
		}

		if transID := r.Header.Get("X-Global-Transaction-ID "); transID != "" {
			entry = entry.WithField("trans_id", transID)
		}

		entry = m.Before(entry, r, remoteAddr)

		if m.logStarting {
			entry.Info("Started handling request")
		}

		rw := NewResponseWriter(w)

		next.ServeHTTP(rw, r)

		latency := m.clock.Since(start)

		m.After(entry, rw, latency, m.Name).Info("Completed handling request")
	})
}

// BeforeFunc is the func type used to modify or replace the *logrus.Entry prior
// to calling the next func in the middleware chain
type BeforeFunc func(*log.Entry, *http.Request, string) *log.Entry

// AfterFunc is the func type used to modify or replace the *logrus.Entry after
// calling the next func in the middleware chain
type AfterFunc func(*log.Entry, ResponseWriter, time.Duration, string) *log.Entry

// DefaultBefore is the default func assigned to *Middleware.Before
func DefaultBefore(entry *log.Entry, req *http.Request, remoteAddr string) *log.Entry {
	return entry.WithFields(log.Fields{
		"request": req.RequestURI,
		"method":  req.Method,
		"remote":  remoteAddr,
	})
}

// DefaultAfter is the default func assigned to *Middleware.After
func DefaultAfter(entry *log.Entry, res ResponseWriter, latency time.Duration, name string) *log.Entry {

	return entry.WithFields(log.Fields{
		"status":      res.Status(),
		"text_status": http.StatusText(res.Status()),
		"took":        latency,
		fmt.Sprintf("measure#%s.latency", name): latency.Nanoseconds(),
	})
}
