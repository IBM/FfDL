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
