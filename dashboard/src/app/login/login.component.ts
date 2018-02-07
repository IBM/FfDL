import { Component, Input, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute, Params } from '@angular/router';
import { EmitterService } from "../shared/services/emitter.service";
import { AuthService } from "../shared/services/auth.service";
import {CookieService, CookieOptions} from "ngx-cookie";

@Component({
  selector: 'login',
  templateUrl: './login.component.html'
})
export class LoginComponent implements OnInit, OnDestroy {

  @Input() endpoint: string;
  @Input() username: string;
  @Input() password: string;

  private cookieService: CookieService;
  private cookieOptions: CookieOptions;

  private environments = [];

  showExpiredLogin: boolean;

  constructor(private router: Router, private authService: AuthService,
      private _cookieService:CookieService, private activatedRoute: ActivatedRoute) {

    EmitterService.get('showNavBar').emit(false);
    EmitterService.get('showExpiredLogin').subscribe((show : boolean) =>{
      this.showExpiredLogin = show;
    });

    this.cookieService = _cookieService;
    this.cookieOptions = {expires: "20"}
  }

  ngOnInit() {
    // inject endpoint/username from query params
    this.activatedRoute.queryParams.subscribe((params: Params) => {
      if(params.endpoint) {
        this.endpoint = params.endpoint;
        if(this.endpoint.indexOf("://") < 0) {
          this.endpoint = "http://" + this.endpoint;
        }
      }
      if(params.username) {
        this.username = params.username;
      }
    });
  }

  login() {
    this.authService.login(this.endpoint, this.username, this.password);
  }

  ngOnDestroy() {}

}
