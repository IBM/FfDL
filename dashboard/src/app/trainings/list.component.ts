import { Component, ViewEncapsulation, OnInit, OnChanges, ElementRef, ViewChild } from '@angular/core';
import {FormBuilder, FormGroup, Validators} from "@angular/forms";
import { DlaasService } from '../shared/services';
import { ModelData, BasicNewModel } from "../shared/models/index";
import { NotificationsService } from 'angular2-notifications';
import { Subscription } from 'rxjs/Subscription';
import { HttpErrorResponse } from "@angular/common/http";
import {CookieService, CookieOptions} from "ngx-cookie";
import {Observable} from "rxjs/Observable";
import {LogLine} from "../shared/models/index";
import 'rxjs/add/operator/share';

interface LastManifestCookie {
  manifest: Blob,
  zipfile: Blob,
}

@Component({
  selector: 'trainings-list',
  templateUrl: './list.component.html',
  styleUrls: ['./list.component.css'],
  encapsulation: ViewEncapsulation.None
})
export class TrainingsListComponent implements OnInit, OnChanges {

    private findSub: Subscription;
    private deleteSub: Subscription;

    trainings: ModelData[];
    trainingsError: Boolean = false;
    trainingId: string;
    current_model: ModelData;
    display_tabs: string = "none";

  constructor(private dlaas: DlaasService,
              private notificationService: NotificationsService,
              private fb: FormBuilder) {
    this.createForm();
  }
  private cookieService: CookieService
  private cookieOptions: CookieOptions

  private lastNewTraining: BasicNewModel;

  private trainingSub: Subscription;

  form: FormGroup;
  formData: FormData;
  zipEvent: HTMLInputElement;
  manifestEvent: HTMLInputElement;
  loading: boolean = false;

  @ViewChild('fileInput') fileInput: ElementRef;

  changeModel(new_model){
    var tab_manager = document.getElementById("tab_manager");
    var back_button = document.getElementById("back_button");
    var jobs_tag = document.getElementById("jobs_tag");
    var training_list = document.getElementById("training_list");

    var training_num;
    for (training_num = 0; training_num < this.trainings.length; training_num ++) {
      if (this.trainings[training_num].model_id == new_model){
        tab_manager.style.display = "";
        back_button.style.display = "";
        jobs_tag.style.display = "none";
        training_list.style.display = "none";

        this.trainingId = new_model;
        this.current_model = this.trainings[training_num];
      }
    }
    this.reformatTime();

    var status_elem = document.getElementById("status_bubble");

    if (this.current_model.training.training_status.status === 'FAILED') {
      status_elem.style.color = "#ee0000";
    } else if (this.current_model.training.training_status.status === 'COMPLETED') {
      status_elem.style.color = "#00aa00"
    } else {
      status_elem.style.color = "#dddd00";
    }
  }

  showTraining() {
    var tab_manager = document.getElementById("tab_manager");
    var back_button = document.getElementById("back_button");
    var jobs_tag = document.getElementById("jobs_tag");
    var training_list = document.getElementById("training_list");

    tab_manager.style.display = "none";
    back_button.style.display = "none";
    jobs_tag.style.display = "";
    training_list.style.display = "";
  }

  reformatTime() {
    var sub_elem = document.getElementById("submission_time");
    var comp_elem = document.getElementById("completion_time");
    var unix_timestamp = parseInt(this.current_model.training.training_status.submitted)
    var d;

    if (unix_timestamp == null){
      sub_elem.innerHTML = "N/A";
    }
    else{
      d = new Date(unix_timestamp)
      sub_elem.innerHTML = d
    }

    unix_timestamp = parseInt(this.current_model.training.training_status.completed)

    if (unix_timestamp == null){
      comp_elem.innerHTML = "N/A";
    }
    else{
      d = new Date(unix_timestamp)
      comp_elem.innerHTML = d
    }
  }

  tabGraphActive() {
    // without this graphs won't resize
    window.dispatchEvent(new Event('resize'));
  }

  createForm() {
    this.form = this.fb.group({
      manifest: null,
      model_definition: null
    });
  }

  status: any = {
    isFirstOpen: true,
    isFirstDisabled: false
  };

  onManifestFileChange(event) {
    this.manifestEvent = event.target;
  }

  onModelzipFileChange(event) {
    this.zipEvent = event.target;
  }

  onSubmit() {
    this.loading = true;
    this.formData = new FormData();
    this.createForm();
    var builtForm = true;

    if(this.manifestEvent.files && this.manifestEvent.files.length > 0
       && this.zipEvent.files && this.zipEvent.files.length > 0) {
      let file = this.manifestEvent.files[0];
      this.formData.append('manifest', file, file.name);
      this.form.get('manifest').setValue({
        filename: file.name,
        filetype: file.type,
      });
      let file2 = this.zipEvent.files[0];
      this.formData.append('model_definition', file2, file2.name);
      this.form.get('model_definition').setValue({
        filename: file2.name,
        filetype: file2.type,
      });
    }
    else {return}

    this.trainingSub = this.dlaas.postTraining(this.formData).subscribe(
      data => {
        this.lastNewTraining = data;
        this.find();
        this.loading = false;
      },
      (err: HttpErrorResponse) => {
        this.loading = false;
        if (err.error instanceof Error) {
          // A client-side or network error occurred. Handle it accordingly.
          console.log('An error occurred:', err.error.message);
        } else {
          // The backend returned an unsuccessful response code.
          // The response body may contain clues as to what went wrong,
          // console.log(`Backend returned code ${err.status}, body was: ${err.error}`);
          console.log("Backend returned: " + String(err));
        }
      }
    );
  }

  clearFile() {
    this.form.get('manifest').setValue(null);
    this.form.get('model_definition').setValue(null);
    this.fileInput.nativeElement.value = '';
  }

  private updateSubscription: Subscription;

  startOngoingUpdate() {
    this.updateSubscription = Observable.interval(1000*20).subscribe(x => {
      this.find();
    });
  }

  ngOnInit() {
    this.find();
    this.startOngoingUpdate();
  }

  ngOnChanges(changes: any) {
    // console.log('ngOnChanges called in training list ')
  }

  ngOnDestroy() {
    this.findSub.unsubscribe();
    if (this.deleteSub) this.deleteSub.unsubscribe();
  }

  find() {
    this.findSub = this.dlaas.getTrainings().subscribe(
      data => { this.trainings = data;
        // console.log(this.trainings)
      },
      err => { this.trainingsError = true; }
    );
  }

  delete(id: String) {
    var isConfirmed = confirm("Are you sure you want to delete " + id + "?");
    if (isConfirmed) {
      this.notificationService.info('Deleting training', 'ID: ' + id);
      this.dlaas.deleteTraining(id).subscribe(
        data => {
          this.notificationService.success('Training deleted.', 'ID: ' + id);
          this.find()
        },
        err => {
          this.notificationService.error('Deletion failed', 'Message: ' + err);
        }
      );
    }
  }

  dotClass(model: ModelData) {
    if (model.training.training_status.status === 'FAILED') {
      return 'red_dot';
    } else if (model.training.training_status.status === 'COMPLETED') {
      return 'green_dot';
    } else {
      return 'yellow_dot';
    }
  }

}
