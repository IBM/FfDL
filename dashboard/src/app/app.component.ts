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
