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
