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

import { Component, OnInit, OnDestroy, ViewEncapsulation } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { Subscription } from 'rxjs/Subscription';

import { DlaasService } from '../shared/services';
import { ModelData } from "../shared/models/index";
import 'rxjs/add/operator/share';

@Component({
    selector: 'training-show',
    templateUrl: './show.component.html',
    // encapsulation: ViewEncapsulation.None
})
export class TrainingsShowComponent implements OnInit, OnDestroy {

    trainingId: string;
    training: ModelData;
    private trainingSub: Subscription;

    constructor(private route: ActivatedRoute, private dlaas: DlaasService) {
    }

    ngOnInit() {
      this.trainingId = this.route.snapshot.params['id'];
      console.log('id: ' + this.trainingId);
      this.trainingSub = this.dlaas.getTraining(this.trainingId).subscribe(t => {
        this.training = t;
      });

    }

    ngOnDestroy() {
      if (this.trainingSub) { this.trainingSub.unsubscribe(); }
    }

    getStatusColor(status: string): string {
      if (status === 'FAILED') {
          return 'text-danger';
      } else if (status === 'COMPLETED') {
          return 'text-success';
      }
    }

    tabGraphActive() {
      // without this graphs won't resize
      window.dispatchEvent(new Event('resize'));
    }

}
