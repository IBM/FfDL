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

import {Component, Input, OnChanges, OnInit} from '@angular/core';
import {DlaasService} from '../shared/services';
import {Subscription} from 'rxjs/Subscription';

import 'rxjs/add/operator/retryWhen';
import 'rxjs/add/operator/delay';
import {EMetrics} from "../shared/models/index";
import {KEY_CODE} from "./logs.component";
import {Observable} from "rxjs/Observable";

@Component({
    selector: 'training-emetrics_raw',
    template:
      `<!-- div *ngIf="showError" class="alert alert-danger" role="alert">
        Oh snap! An error occurred while loading the available log data!
      </div -->
      <!--<sk-fading-circle [isRunning]="showSpinner"></sk-fading-circle>-->
    <div class="col-md-12" style="margin-bottom: 2px">
      <div class="btn-group" data-toggle="buttons">
        <label ngbButtonLabel style="vertical-align: middle; margin-top: 5px; margin-right: 3em">
          <input type="checkbox" ngbButton [(ngModel)]="follow" (click)="followEvent()" style="vertical-align: middle;" name="follow">&nbsp;&nbsp;Follow&nbsp;&nbsp;
        </label>
        <button class="button" (click)="home()">Home</button>
        <button class="button" (click)="end()">End</button>
        <button class="button" (click)="pageUp()">PgUp</button>
        <button class="button" (click)="pageDown()">PgDn</button>
      </div>
      <pre *ngIf="!showSpinner && !showError" (keydown)="keyEvent($event)" tabindex="0">
      <div id="box" style="overflow: scroll; height: 600px;">
        <table>
           <tr>
             <th scope="row">rindex&nbsp;&nbsp;</th>
             <th scope="row">time</th>
             <th scope="row">label&nbsp;&nbsp;</th>
             <th *ngFor="let key of eTimeKeys" scope="row">{{key}}&nbsp;&nbsp;</th>
             <th *ngFor="let key of emValueKeys" scope="row">{{key}}&nbsp;&nbsp;</th>
           </tr>
           <tbody *ngFor="let t of emetrics">
              <tr>
                <td>{{t.meta.rindex}}&nbsp;&nbsp;</td>
                <td>{{t.meta.time}}&nbsp;&nbsp;</td>
                <td>{{t.grouplabel}}&nbsp;&nbsp;</td>
                <td *ngFor="let key of eTimeKeys">{{t.etimes[key].value}}&nbsp;&nbsp;</td>
                <td *ngFor="let key of emValueKeys">{{t.values[key].value}}&nbsp;&nbsp;</td>
              </tr>
            </tbody>
          </table>
        </div>
      </pre>
    </div>`,
    styleUrls: ['./emetricsraw.component.css']
})
export class TrainingEMetricsRawComponent implements OnChanges {

  private _trainingId: string;

  private emetrics: EMetrics[];
  private logsError: Boolean = false;

  private pagesize: number = 20;
  private relativePageIncrement = -this.pagesize;
  private pos: number = -1;

  private home_pos: number = 0;
  private end_pos: number = -1;

  private prevTime: string = "";

  private findSub: Subscription;

  public eTimeKeys: string[];
  public emValueKeys: string[];

  private subscription: Subscription;

  showSpinner = false;
  showError = false;

  public follow: boolean = false

  // constructor(private dlaas: DlaasService,
  //             private notificationService: NotificationsService) { }

  constructor(private dlaas: DlaasService) {
  }

  @Input()
  set trainingId(trainId: string){
    this._trainingId = trainId;
    this.pos = -1
    this.update();
  }

  ngOnChanges(changes: any) {
    // console.log('ngOnChanges called in training list ')
  }

  ngOnDestroy() {
    if (this.subscription) { this.subscription.unsubscribe(); }
  }

  update() {
    this.find(this.pos, this.pagesize, "");
  }

  decrement() {
    this.pos -= 1;
    if(this.relativePageIncrement >= 0) {
      if(this.pos < 0) {
        this.pos = 0;
    }
    }

    this.update()
  }

  increment() {
    this.pos += 1;
    if(this.relativePageIncrement < 0) {
      if(this.pos >= 0) {
        this.pos = -1
      }
    }
    this.update()
  }

  pageUp() {
    this.pos -= this.pagesize;
    if(this.relativePageIncrement >= 0) {
      if (this.pos < 0) {
        this.pos = 0
      }
    }

    this.update()
  }

  pageDown() {
    this.pos += this.pagesize;
    if(this.relativePageIncrement < 0) {
      if(this.pos >= 0) {
          this.pos = -1
      }
    }
    this.update()
  }

  home() {
    this.pos = this.home_pos;
    this.relativePageIncrement = this.pagesize;
    this.update()
    var element = document.getElementById('box');
    element.scrollTop = 0;
  }

  end() {
    this.pos = this.end_pos;
    this.relativePageIncrement = -this.pagesize;
    this.update()
    var element = document.getElementById('box');
    element.scrollTop = element.scrollHeight;
  }

  // @HostListener('window:keyup', ['$event'])
  keyEvent(event: KeyboardEvent) {
    // console.log(event);
    if (document.hidden) {
      return
    }

    if (event.keyCode === KEY_CODE.PAGE_DOWN) {
      this.pageDown();
    }
    if (event.keyCode === KEY_CODE.PAGE_UP) {
      this.pageUp();
    }
    if (event.keyCode === KEY_CODE.END) {
      this.end();
    }
    if (event.keyCode === KEY_CODE.HOME) {
      this.home();
    }
    if (event.keyCode === KEY_CODE.UP_ARROW) {
      this.decrement();
    }
    if (event.keyCode === KEY_CODE.DOWN_ARROW) {
      this.increment();
    }
  }

  sleep(ms = 0) {
    return new Promise(r => setTimeout(r, ms));
  }

  startOngoingUpdate() {
    this.subscription = Observable.interval(1000*4).subscribe(x => {
      for (; true; ) {
        let origPrevTime = this.prevTime;
        this.end();
        if(this.prevTime == origPrevTime) {
          break
        }
        this.sleep(10)
      }
    });
  }

  stopOngoingUpdate() {
    if (this.subscription) {
      this.subscription.unsubscribe();
      this.subscription = null
    }
  }

  followEvent() {
    if(!this.follow) {
      this.startOngoingUpdate()
    } else {
      this.stopOngoingUpdate()
    }
  }

  private find(pos: number, pagesize: number, since: string) {
    // this.findSub = this.dlaas.getTrainingMetrics(this._trainingId, pos, pagesize, since).subscribe(
    this.findSub = this.dlaas.getTrainingMetrics(this._trainingId, pos, 999, since).subscribe(
      data => {
        this.emetrics = data;
        if (this.emetrics.length == 0) {
          return;
        }
        this.eTimeKeys = Object.keys(this.emetrics[0].etimes);
        this.emValueKeys = Object.keys(this.emetrics[0].values);
      },
      err => {
        this.logsError = true;
      }
    );
  }


}
