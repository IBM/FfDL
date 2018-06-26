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

import { Component, ViewEncapsulation, OnInit } from '@angular/core';
import { Router, NavigationEnd } from '@angular/router';
import { AppService } from './shared/services';
import {CookieService} from "ngx-cookie";

// var $ = require("jquery");

@Component({
  selector: 'my-app',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css'],
  encapsulation: ViewEncapsulation.None
})
export class AppComponent implements OnInit {

  constructor(private router: Router, private appService: AppService, private _cookieService:CookieService) {
  }

  getCookie(key: string){
    return this._cookieService.get(key);
  }

  putCookie(key: string, val: string){
    return this._cookieService.put(key, val);
  }

  ngOnInit() {
    this.router.events.subscribe(navEvt => {
      if (navEvt instanceof NavigationEnd) {
        this.appService.inferTitleFromUrl(navEvt.urlAfterRedirects);
      }
    });
  }

}
