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
