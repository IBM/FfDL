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

import { Routes } from '@angular/router';
// import { TrainingsListComponent, TrainingsShowComponent } from './trainings';
import { TrainingsListComponent } from "./trainings/list.component";
import { TrainingsShowComponent } from "./trainings/show.component";
import { ModelsCreateComponent } from "./models/create.component";
import { AnalyticsMainComponent } from "./analytics/main.component";
import { LoginComponent } from "./login/login.component";
import { AuthGuard } from "./shared/services/auth-guard.service";

export const ROUTES: Routes = [
    {
        path: '',
        redirectTo: '/trainings/list',
        pathMatch: 'full'
    },
    {
        path: 'login',
        component: LoginComponent,
    },
    {
      path: 'models/create',
      component: ModelsCreateComponent,
      canActivate: [AuthGuard]
    },
    {
        path: 'trainings/list',
        component: TrainingsListComponent,
        canActivate: [AuthGuard]
    },
    {
        path: 'trainings/:id/show',
        component: TrainingsShowComponent,
        canActivate: [AuthGuard]
    },
    {
        path: 'analytics/main',
        component: AnalyticsMainComponent,
        canActivate: [AuthGuard]
    },

];
