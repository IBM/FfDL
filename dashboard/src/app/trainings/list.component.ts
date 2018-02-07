import { Component, ViewEncapsulation, OnInit, OnChanges, ElementRef, ViewChild } from '@angular/core';
import {FormBuilder, FormGroup, Validators} from "@angular/forms";
import { DlaasService } from '../shared/services';
import { ModelData, BasicNewModel } from "../shared/models/index";
import { NotificationsService } from 'angular2-notifications';
import { Subscription } from 'rxjs/Subscription';
import { HttpErrorResponse } from "@angular/common/http";
import {CookieService, CookieOptions} from "ngx-cookie";
import {Observable} from "rxjs/Observable";

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
  formData: FormData = new FormData();
  loading: boolean = false;

  @ViewChild('fileInput') fileInput: ElementRef;

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
    if(event.target.files && event.target.files.length > 0) {
      let file = event.target.files[0];
      this.formData.append('manifest', file, file.name);
      this.form.get('manifest').setValue({
        filename: file.name,
        filetype: file.type,
      });
    }
  }

  onModelzipFileChange(event) {
    if(event.target.files && event.target.files.length > 0) {
      let file = event.target.files[0];
      this.formData.append('model_definition', file, file.name);
      this.form.get('model_definition').setValue({
        filename: file.name,
        filetype: file.type,
      });
    }
  }

  onSubmit() {
    this.loading = true;

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
    this.startOngoingUpdate()
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

  getStatusColor(model: ModelData): string {
    if (model.training.training_status.status === 'FAILED') {
      return 'table-danger';
    } else if (model.training.training_status.status === 'COMPLETED') {
      return 'table-success';
    }
  }

}
