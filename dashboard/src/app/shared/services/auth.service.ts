import { Injectable } from '@angular/core';
import { Router } from '@angular/router';

import {SessionStorageService} from 'ngx-webstorage';
import { Observable } from 'rxjs/Observable';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import {EmitterService} from './emitter.service';
import { LoginData } from '../models/index';

const tokenKey = 'id_token';
const tokenExpirationKey = 'token_expiration';
const usernameKey = 'username';
const environmentKey = 'environment';
const roleKey = 'role';

const ROLE_USER = 'user';
const ROLE_ADMIN = 'admin';

@Injectable()
export class AuthService {

  constructor(private router: Router, private http: HttpClient, private storage: SessionStorageService) {
  }

  login(environment:string, username:string, password:string) {
    // this.getWatsonToken(environment, username, password).subscribe(
    //   (token:string) => {
    const token = "abcdef";
    this.storage.store(tokenKey, token);
    this.storage.store(usernameKey, username);
    this.storage.store(environmentKey, environment);

    // TODO replace this with proper RBAC logic!!
    let role = username == 'admin' ? ROLE_ADMIN : ROLE_USER;
    this.storage.store(roleKey, role);

    // calcuate expiration date -  add 59 minutes - 60 is the token expiry by the server
    let now = new Date(Date.now());
    let expirationDate = new Date();
    expirationDate.setTime(now.getTime() + (59 * 60 * 1000)); // 59 min in msec
    this.storage.store(tokenExpirationKey, expirationDate.toUTCString());

    EmitterService.get('login_success').emit(
      new LoginData(environment, username, token, expirationDate.toUTCString(), role));

    this.router.navigateByUrl('');
    //   },
    //   (err:any) => {
    //     console.log('login failed');
    //     // TODO test this
    //     EmitterService.get('login_failed').emit('login failed for some reason');
    //   }
    // );

  }

  logout() {
    // To log out, just remove the token and profile
    this.storage.clear(tokenKey);
    this.storage.clear(tokenExpirationKey);
    this.storage.clear(usernameKey);
    this.storage.clear(environmentKey);

    // Send the user back to the public deals page after logout
    this.router.navigateByUrl('/login');
  }

  loggedIn(): boolean {
    // logged in means token present and not expired
    let token = this.storage.retrieve(tokenKey);
    let username = this.storage.retrieve(usernameKey);
    let expiration = this.storage.retrieve(tokenExpirationKey);
    let environment = this.storage.retrieve(environmentKey);

    if (token != null && username != null && environment != null && expiration != null) {
      let now = new Date(Date.now());
      let expDate = new Date(expiration);
      if (now.getTime() <= expDate.getTime()) {
        return true;
      } else {
        EmitterService.get('showExpiredLogin').emit(true); // login expired
      }
    }
    return false;
  }

  getLoginDataFromSession(): LoginData {
    let token = this.storage.retrieve(tokenKey);
    let username = this.storage.retrieve(usernameKey);
    let expiration = this.storage.retrieve(tokenExpirationKey);
    let environment = this.storage.retrieve(environmentKey);
    let role = this.storage.retrieve(roleKey);
    return new LoginData(environment, username, token, expiration, role);
 }

  private getWatsonToken(environment: String, username: String, password: String): Observable<String> {
    let authEndpoint = "/token";

    // if we are developing locally validate and CSF dev token endpoint
    if (environment === 'mynamespace') {
      authEndpoint = '/token-namespace';
    } else if ((environment === 'local') || (environment == 'development')) {
     authEndpoint = '/token-development';
    } else if (environment === 'staging') {
     authEndpoint = '/token-staging';
    }
    // create headers
    let headers = new HttpHeaders();
    headers.append('Authorization', 'Basic ' + btoa(username + ':' + password));
    headers.append('Accept', 'text/plain');

    return this.http.get(authEndpoint, { headers: headers, observe: "response" })
       .map(response => { // TODO handle error properly
          if (response.status === 401) {
            console.log("status 401");
          }
          if (response.status === 200) {
           return response.body
          }
       }).catch((error: any) => Observable.throw(error.text() || 'Server error'));
  }

  isAdmin() {
    return this.role() == ROLE_ADMIN;
  }

  role(): string {
    if(!this.loggedIn()) {
      return null;
    }
    return this.getLoginDataFromSession().role;
  }

  user(): string {
    if(!this.loggedIn()) {
      return null;
    }
    return this.getLoginDataFromSession().username;
  }

}
