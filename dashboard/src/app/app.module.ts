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

import { NgModule, ApplicationRef } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ReactiveFormsModule } from '@angular/forms';
import { BrowserModule, Title } from '@angular/platform-browser';
import { HttpClientModule } from '@angular/common/http';
import { RouterModule } from '@angular/router';

import { TabsModule, AccordionModule } from "ngx-bootstrap";
import { AppService, DlaasService } from './shared/services';
import { AppComponent } from './app.component';
import { ChartsModule } from 'ng2-charts/ng2-charts';
import { LoginComponent } from "./login/login.component";

import { ROUTES } from './app.routes';
import { MenuComponent } from './menu';
import { ModelsListComponent } from "./models/list.component";
import { ModelsCreateComponent } from "./models/create.component";
import { TrainingsListComponent, TrainingsShowComponent} from './trainings';
import { TrainingEMetricsComponent } from "./trainings/emetrics.component";
import { TrainingEMetricsRawComponent } from "./trainings/emetricsraw.component";
import { TrainingLogsComponent } from "./trainings/logs.component";
import { AnalyticsMainComponent } from "./analytics/main.component";
import { ProfileListComponent } from "./profile/list.component";
import { ProfileShowComponent } from "./profile/show.component";
import { SimpleNotificationsModule } from "angular2-notifications";
import { Ng2Webstorage } from "ngx-webstorage";
import { AuthService} from "./shared/services/auth.service";
import { AuthGuard } from "./shared/services/auth-guard.service";
import { EmitterService } from "./shared/services/emitter.service";
import { Autofocus } from "./shared/directives/autofocus";

// import { NG2D3Module } from "ng2d3";
// import {SpinnerModule} from "angular2-spinner/dist";
import {CookieModule, CookieOptions, CookieService} from "ngx-cookie";

import { BrowserAnimationsModule } from "@angular/platform-browser/animations";

@NgModule({
    declarations: [
      AppComponent,
      MenuComponent,
      ModelsListComponent,
      ModelsCreateComponent,
      LoginComponent,
      TrainingsListComponent,
      TrainingsShowComponent,
      TrainingLogsComponent,
      TrainingEMetricsRawComponent,
      TrainingEMetricsComponent,
      // FadingCircleComponent,
      AnalyticsMainComponent,
      ProfileListComponent,
      ProfileShowComponent,
      Autofocus
    ],
    imports: [
      BrowserModule,
      FormsModule,
      ReactiveFormsModule,
      HttpClientModule,
      // Ng2BootstrapModule,
      ChartsModule,
      RouterModule.forRoot(ROUTES, { useHash: true }),
      TabsModule.forRoot(),
      AccordionModule.forRoot(),
      SimpleNotificationsModule.forRoot(),
      Ng2Webstorage,
      // NG2D3Module,
      BrowserAnimationsModule,
      // SpinnerModule,
      CookieModule.forRoot(),
    ],
    providers: [
      Title,
      AppService,
      AuthService,
      AuthGuard,
      DlaasService,
      EmitterService,
      CookieService,
    ],
    bootstrap: [
      AppComponent
    ]
})
export class AppModule {
}
