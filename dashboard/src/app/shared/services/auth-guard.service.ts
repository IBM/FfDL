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
import {ActivatedRouteSnapshot, Router, RouterStateSnapshot} from '@angular/router';
import { CanActivate } from '@angular/router';
import { AuthService } from './auth.service';
import { EmitterService } from './emitter.service';

@Injectable()
export class AuthGuard implements CanActivate {

  constructor(private auth: AuthService, private router: Router) {}

  canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot) {
    // If user is not logged in we'll send them to the homepage
    if (!this.auth.loggedIn()) {
      this.router.navigate(['/login']);
      return false;
    }

    if (!this.auth.isAdmin()) {
      // TODO temporary code, should not be hardcoded in here! Move to separate RBAC service class.
      if (route.url.length > 0 && route.url[0].path == 'analytics') {
        this.router.navigate(['/']);
        return false;
      }
    }

    EmitterService.get('showNavBar').emit(true)
    return true;
  }

}
