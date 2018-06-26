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

import {Component, HostListener, Input, OnChanges, OnInit} from '@angular/core';
import {DlaasService} from '../shared/services';
import {Subscription} from 'rxjs/Subscription';

import 'rxjs/add/operator/retryWhen';
import 'rxjs/add/operator/delay';
import 'rxjs/add/observable/interval';

import {LogLine} from "../shared/models/index";
import {Observable} from "rxjs/Observable";

export enum KEY_CODE {
  PAGE_UP = 33,
  PAGE_DOWN = 34,
  END = 35,
  HOME = 36,
  LEFT_ARROW = 37,
  UP_ARROW = 38,
  RIGHT_ARROW = 39,
  DOWN_ARROW = 40
}

@Component({
    selector: 'training-logs',
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
        <button (click)="home()">Home</button>
        <button (click)="end()">End</button>
        <button (click)="pageUp()">PgUp</button>
        <button (click)="pageDown()">PgDn</button>
        <button (click)="decrement()">Up</button>
        <button (click)="increment()">Dn</button>
      </div>
      <pre *ngIf="!showSpinner && !showError"  (keydown)="keyEvent($event)" tabindex="0">
        <!--<span *ngFor="let line of logs">{{line.line}}</span>-->
        <table>
          <tbody *ngFor="let t of logs">
            <tr>
              <td (keydown)="keyEvent($event)">{{t.meta.rindex}}&nbsp;&nbsp;</td>
              <td>{{t.meta.time}}&nbsp;&nbsp;</td>
              <td>{{t.line}}</td>
            </tr>
          </tbody>
        </table>
      </pre>
    </div>`,
    styleUrls: ['./logs.component.css']
})
export class TrainingLogsComponent implements OnInit, OnChanges {

@Input() private trainingId: string;

  private logs: LogLine[];
  private logsError: Boolean = false;

  private pagesize: number = 20;
  private relativePageIncrement = -this.pagesize;
  private pos: number = -1;

  private home_pos: number = 0;
  private end_pos: number = -1;

  private prevTime: string = "";

  private findSub: Subscription;

  private subscription: Subscription;

  showSpinner = false;
  showError = false;

  public follow: boolean = false;

  constructor(private dlaas: DlaasService) {
  }

  ngOnChanges(changes: any) {
    // console.log('ngOnChanges called in training list ')
  }

  ngOnInit() {
    this.find(this.pos, this.pagesize, "");
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
        this.pos = 0
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
      if(this.pos < 0) {
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
  }

  end() {
    this.pos = this.end_pos;
    this.relativePageIncrement = -this.pagesize;
    this.update()
  }

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
    // console.log("Inside followEvent() follow="+this.follow)
    if(!this.follow) {
      this.startOngoingUpdate()
    } else {
      this.stopOngoingUpdate()
    }
  }

  private find(pos: number, pagesize: number, since: string) {
    this.findSub = this.dlaas.getTrainingLogs(this.trainingId, pos, pagesize, since).subscribe(
      data => {
        // console.log("pos: "+pos+", pageszie: "+pagesize+", since:"+since)
        this.logs = data;
        if (this.logs.length == 0) {
          return;
        }
        this.prevTime = this.logs[this.logs.length - 1].meta.time
      },
      err => {
        this.logsError = true;
      }
    );
  }


}
