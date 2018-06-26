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

import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs/Observable';
import { ModelData, LoginData, EMetrics, LogLine, BasicNewModel } from '../models';
import { EmitterService } from "./emitter.service";
import { AuthService } from "./auth.service";

import 'rxjs/add/operator/catch';
import 'rxjs/add/observable/throw';
import 'rxjs/add/operator/map';
import 'rxjs/add/observable/dom/webSocket';

declare const Buffer

// TODO make configurable
const ANALYTICS_API_URL = 'http://localhost:30010/v1/analytics';

@Injectable()
export class DlaasService {

  private endpoint: string;
  private loginData: LoginData;

  constructor(private http: HttpClient, private auth: AuthService) {
    // read data from session
    this.loginData = this.auth.getLoginDataFromSession();
    this.setEndpointForEnv();

    // react on login event
    EmitterService.get('login_success').subscribe( (data:LoginData) => {
      this.loginData = data;
      this.setEndpointForEnv();
    });
  }

  // get all trainings in DLaaS
  getTrainings(): Observable<ModelData[]> {
    // create the request, store the `Observable` for subsequent subscribers
    let headers: HttpHeaders = this.getHeaders();
    let trainings: Observable<ModelData[]> = this.http.get(this.url('/v1/models?version=2017-02-13'),
      { headers: headers, observe: "response" })
      .map(response => {
        // console.log('have response for getTrainings: '+JSON.stringify(response, null, 4))
        // console.log('have response for getTrainings.status: '+response.status)
        if (response.status === 200) {
          return response.body["models"];
        }
      })
      .catch((error: any) => Observable.throw(error.json().error || 'Server error'));
    return trainings;
  }

  postTraining(formData: FormData): Observable<BasicNewModel> {
    let headers: HttpHeaders = this.getHeaders();
    let postRequest: Observable<BasicNewModel> = this.http.post(this.url('/v1/models?version=2017-02-13'),
      formData, {
        headers: headers,
        observe: "response"
      })
      .map(response => {
        if (response.status === 200) {
          return response.body;
        }
      })
      .catch((err: any) => Observable.throw(err.error.message || 'Server error'));
    return postRequest;
  }

  getTraining(id: String): Observable<ModelData> {
    // create the request, store the `Observable` for subsequent subscribers
    return this.http.get(this.url('/v1/models/') + id + '?version=2017-02-13',
      { headers: this.getHeaders(), observe: "response" })
      .map(response => {
        if (response.status === 200) {
          return response.body;
        }
        // TODO handle 404
      })
      .catch((error: any) => Observable.throw(error.body.error || 'Server error'));
  }

  getTrainingLogs(id: string, pos: number, pagesize: number, since: string): Observable<LogLine[]> {
    // create the request, store the `Observable` for subsequent subscribers
    return this.http.get(this.url('/v1/logs/') + id +
      '/loglines?version=2017-02-13"+' +
      '&pos='+pos+
      '&pagesize='+pagesize+
      '&since_time='+since, { headers: this.getHeaders(), observe: "response" })
      .map(response => {
        if (response.status === 200) {
          return response.body["models"];
        }
      })
      .catch((error: any) => Observable.throw(error.json().error || 'Server error'));
  }

  getTrainingMetrics(id: string, pos: number, pagesize: number, since: string): Observable<EMetrics[]> {
    // create the request, store the `Observable` for subsequent subscribers
    return this.http.get(this.url('/v1/logs/') + id +
      '/emetrics?version=2017-02-13"+' +
      '&pos='+pos+
      '&pagesize='+pagesize+
      '&since_time='+since, { headers: this.getHeaders(), observe: "response" })
      .map(response => {
        if (response.status === 200) {
          return response.body["models"];
        }
      })
      .catch((error: any) => Observable.throw(error.json().error || 'Server error'));
  }

  deleteTraining(id: String): Observable<boolean> {
    // create the request, store the `Observable` for subsequent subscribers
    return this.http.delete(this.url('/v1/models/') + id + '?version=2017-02-13',
      { headers: this.getHeaders(), observe: "response" })
      .map(response => {
        if (response.status === 200) {
          return true
        }
        // TODO handle 404
        // make it shared so more than one subscriber can get the result
      })
      .catch((error: any) => Observable.throw(error.text() || 'Server error'));
  }

  getTrainingJobs(sources: string[]): Observable<any[]> {
    var payload = {'variables': 123, 'sources': sources};
    var headers = this.getHeaders();
    headers.append('content-type', 'application/json');
    return this.http.post(ANALYTICS_API_URL, JSON.stringify(payload),
      { headers: headers, observe: "response" })
      .map(response => {
        if (response.status === 200) {
          return response.body;
        }
      })
      .catch((error: any) => Observable.throw(error.json().error || 'Server error'));
  }

  private resultSelector(e: MessageEvent): string {
    return e.data;
  }

  private setEndpointForEnv(): void {
    // this.endpoint = '/';
    if (this.loginData.environment === 'local') {
      this.endpoint = '/local';
    } else if (this.loginData.environment == 'development') {
      this.endpoint = '/development';
    } else if (this.loginData.environment === 'staging') {
      this.endpoint = '/staging';
    } else {
      this.endpoint = this.loginData.environment
      // this.endpoint = '/mynamespace';
      // this.loginData.username
    }
  }

  private getHeaders() : HttpHeaders {
    let headers = new HttpHeaders();
    if (this.loginData.environment === 'local' || this.loginData.environment === 'mynamespace' || this.loginData.environment.startsWith("http")) {
    // if (this.loginData.environment === 'local') {
      headers = headers.append('X-Watson-Userinfo', 'bluemix-instance-id=' + this.loginData.username);
      headers = headers.append('Authorization', 'Basic ' + btoa(this.loginData.username + ':dummy'));
    }
    headers = headers.append('X-Watson-Authorization-Token', this.loginData.token)
    return headers;
  }

  private url(path: string): string {
    // console.log('path: ', this.endpoint + path)
    return this.endpoint + path;
  }

}
