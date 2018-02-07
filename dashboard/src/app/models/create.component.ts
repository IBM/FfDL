import { Component, OnInit, OnDestroy, ViewEncapsulation } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs/Subscription';
import { DlaasService } from '../shared/services';
import { ModelData } from "../shared/models/index";
import 'rxjs/add/operator/share';

@Component({
  selector: 'my-models-create',
  templateUrl: './create.component.html',
  styleUrls: ['./create.component.css'],
  // encapsulation: ViewEncapsulation.None
})
export class ModelsCreateComponent implements OnInit, OnDestroy {

  private training: ModelData;

  private trainingSub: Subscription;

  public manifestFilePath: string = ""

  status: any = {
    isFirstOpen: true,
    isFirstDisabled: false
  };

  constructor(private route: ActivatedRoute, private dlaas: DlaasService) {
  }

  ngOnInit() {

  }

  ngOnDestroy() {
    if (this.trainingSub) { this.trainingSub.unsubscribe(); }
  }

  public onFileSelect() {
    console.log(this.manifestFilePath)
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
