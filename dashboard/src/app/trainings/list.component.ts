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

import { Component, Inject, ViewEncapsulation, OnInit, OnChanges, ElementRef, ViewChild } from '@angular/core';
import { FormControl, FormBuilder, FormGroup, Validators} from "@angular/forms";
import { DlaasService } from '../shared/services';
import { ModelData, BasicNewModel } from "../shared/models/index";
import { NotificationsService } from 'angular2-notifications';
import { Subscription } from 'rxjs/Subscription';
import { HttpErrorResponse, HttpClient, HttpHeaders } from "@angular/common/http";
import {CookieService, CookieOptions} from "ngx-cookie";
import {Observable} from "rxjs/Observable";
import {LogLine} from "../shared/models/index";
import {MatDialog, MatDialogRef, MAT_DIALOG_DATA} from '@angular/material';
import { AuthService } from "../shared/services/auth.service";
import { EmitterService } from "../shared/services/emitter.service";
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
    private loginData;

    trainings: ModelData[];
    trainingsError: Boolean = false;
    art_check: Boolean = false;
    fairness_check: Boolean = false;
    deployment: Boolean = false;
    trainingId: string;
    artTrainingID: string;
    aifTrainingID: string;
    current_model: ModelData;
    art_current_model: ModelData;
    aif_current_model: ModelData;
    display_tabs: string = "none";
    metrics: string[];
    f_metrics: string[];
    deployment_status: string = "";
    deployment_url: string = "";
    on_pipeline_page: Boolean = false;


  constructor(private dlaas: DlaasService,
              private notificationService: NotificationsService,
              private fb: FormBuilder,
              public dialog: MatDialog,
              private http: HttpClient,
              private auth: AuthService) {
    this.createForm();
    // read data from session
    this.loginData = this.auth.getLoginDataFromSession();

    // react on login event
    EmitterService.get('login_success').subscribe( (data) => {
      this.loginData = data;
    });
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
    var pipeline_manager = document.getElementById("pipeline_manager");

    var training_num;
    for (training_num = 0; training_num < this.trainings.length; training_num ++) {
      if (this.trainings[training_num].model_id == new_model){
        tab_manager.style.display = "";
        back_button.style.display = "";
        pipeline_manager.style.display = "none";
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
      status_elem.style.color = "#00aa00";
    } else {
      status_elem.style.color = "#dddd00";
    }
  }

  changePipeline(new_model){
    this.on_pipeline_page = true;
    var tab_manager = document.getElementById("tab_manager");
    var pipeline_manager = document.getElementById("pipeline_manager");
    var jobs_tag = document.getElementById("jobs_tag");
    var training_list = document.getElementById("training_list");
    var back_button = document.getElementById("back_button");
    back_button.style.display = "";
    pipeline_manager.style.display = "";
    tab_manager.style.display = "none";
    jobs_tag.style.display = "none";
    training_list.style.display = "none";
    this.artUpdate(new_model);
    this.deploymentUpdate(new_model);
  }

  artUpdate(new_model){
    this.art_check = false;
    this.fairness_check = false;
    var training_num;
    for (training_num = 0; training_num < this.trainings.length; training_num ++) {
      if (this.trainings[training_num].model_id == new_model){
        this.trainingId = new_model;
        this.current_model = this.trainings[training_num];
        }
      if (this.trainings[training_num].name == "robustnesscheck_" + new_model){
          this.artTrainingID = this.trainings[training_num].model_id;
          this.art_current_model = this.trainings[training_num];
          this.art_check = true;
          var status_elem_art_result = document.getElementById("art_result");
          status_elem_art_result.style.display = "";
          var status_elem_art = document.getElementById("status_bubble_art");
          if (this.art_current_model.training.training_status.status === 'FAILED') {
            status_elem_art.style.color = "#ee0000";
          } else if (this.art_current_model.training.training_status.status === 'COMPLETED') {
            status_elem_art.style.color = "#00aa00";
            this.dlaas.getTrainingLogs(this.artTrainingID, -1, 20, "").subscribe(
              data => {
                for(var logline = 0; logline < data.length; logline++){
                if(data[logline]['line'].includes("metrics:")){
                  var core_metric = data[logline]['line'].split("{")[1].split("}")[0].replace(/'/g,"")
                  this.metrics = core_metric.split(",");
                  break;
                }
              }
              },
              err => {
                alert("There's an error loading the metrics: Robustness Job or logs not found.")
              }
            );
          } else {
            status_elem_art.style.color = "#dddd00";
          }
      }
      if (this.trainings[training_num].name == "fairnesscheck_" + new_model){
        this.fairness_check = true;
        this.aifTrainingID = this.trainings[training_num].model_id;
        this.aif_current_model = this.trainings[training_num];
        var status_elem_aif_result = document.getElementById("aif_result");
        status_elem_aif_result.style.display = "";
        var status_elem_aif = document.getElementById("status_bubble_aif");
        if (this.aif_current_model.training.training_status.status === 'FAILED') {
          status_elem_aif.style.color = "#ee0000";
        } else if (this.aif_current_model.training.training_status.status === 'COMPLETED') {
          status_elem_aif.style.color = "#00aa00";
          this.dlaas.getTrainingLogs(this.aifTrainingID, -1, 20, "").subscribe(
            data => {
              for(var logline = 0; logline < data.length; logline++){
              if(data[logline]['line'].includes("metrics:")){
                var core_metric = data[logline]['line'].split("{")[1].split("}")[0].replace(/'/g,"")
                this.f_metrics = core_metric.split(",");
                break;
              }
            }
            },
            err => {
              alert("There's an error loading the metrics: Fairness Job or logs not found.")
            }
          );
        } else {
          status_elem_aif.style.color = "#dddd00";
        }
      }
    }
    if(this.art_check == false){
      var status_elem_art_result = document.getElementById("art_result");
      status_elem_art_result.style.display = "none";
    }
    if(this.fairness_check == false){
      var status_elem_aif_result = document.getElementById("aif_result");
      status_elem_aif_result.style.display = "none";
    }
    this.reformatTime();

    var status_elem = document.getElementById("status_bubble_p");

    if (this.current_model.training.training_status.status === 'FAILED') {
      status_elem.style.color = "#ee0000";
    } else if (this.current_model.training.training_status.status === 'COMPLETED') {
      status_elem.style.color = "#00aa00";
    } else {
      status_elem.style.color = "#dddd00";
    }
  }

  deploymentUpdate(new_model){
    this.deployment = false;
    let header_json = {"Content-Type":"application/json"};
    let headers = new HttpHeaders(header_json);
    var formData = {
      "public_ip": "169.48.165.244",
      "deployment_name": new_model,
      "training_id": new_model,
      "check_status_only": "true"
    };
    var sync_elem = document.getElementById("sync");
    sync_elem.className = "fa fa-refresh fa-spin";
    this.http.get("http://deployment-python.default.aisphere.info", { headers: headers, params: formData })
          .subscribe(data => {
          console.log(data);
          this.deployment_status = data['deployment_status'];
          if(this.deployment_status != undefined && this.deployment_status != "NONE" && this.deployment_status != "UNKNOWN" && this.deployment_status != "CREATING"){
            this.deployment = true;
            var status_elem = document.getElementById("status_bubble_deployment");
            if (this.deployment_status === 'ERROR') {
              status_elem.style.color = "#ee0000";
            } else if (this.deployment_status === 'READY' || this.deployment_status === 'AVAILABLE') {
              status_elem.style.color = "#00aa00";
            } else {
              status_elem.style.color = "#dddd00";
            }
            this.deployment_url = data['deployment_url'];
          }
          var status_elem_deploy_result = document.getElementById("deployment_result");
          var status_deploy_buttom = document.getElementById("deploy_button");
          if(this.deployment == false){
            status_elem_deploy_result.style.display = "none";
            status_deploy_buttom.style.display = "";
          }else{
            status_elem_deploy_result.style.display = "";
            status_deploy_buttom.style.display = "none";
          }
          var sync = document.getElementById("sync");
          sync.className = "fa fa-refresh";
        }, error => {
          console.log(error);
          var sync = document.getElementById("sync");
          sync.className = "fa fa-refresh";
        });
  }

  showTraining() {
    this.on_pipeline_page = false;
    var tab_manager = document.getElementById("tab_manager");
    var back_button = document.getElementById("back_button");
    var jobs_tag = document.getElementById("jobs_tag");
    var training_list = document.getElementById("training_list");
    var pipeline_manager = document.getElementById("pipeline_manager");

    tab_manager.style.display = "none";
    back_button.style.display = "none";
    jobs_tag.style.display = "";
    training_list.style.display = "";
    pipeline_manager.style.display = "none";
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

  ARTForm(){
    var cpu = 10;
    var gpu = 0;
    var memory = 10;
    var learner = 1;
    var name = this.current_model.name
    var art_function_link = "https://openwhisk.ng.bluemix.net/api/v1/web/ckadner_org_dev/default/robustness_check.json"
      const dialogRef = this.dialog.open(ArtDialog, {
        height: '600px',
        data: { name: name, cpu: cpu, gpu: gpu,
                memory: memory, learner: learner,
                auth_url: "https://s3-api.us-geo.objectstorage.softlayer.net",
                training_data: training_data,
                training_result: training_result,
                user_name: user_name,
                password: password
              }
      });

      dialogRef.afterClosed().subscribe(result => {
        if (result == undefined){
        }else{
          let header_json = {"Content-Type":"application/json", "X-Require-Whisk-Auth":"fiddle"};
          let headers = new HttpHeaders(header_json);
          var formData = {
            "ffdl_service_url": this.loginData['environment'],
            "basic_authtoken": "test",
            "watson_auth_token": "bluemix-instance-id=test-user",
            "aws_endpoint_url": result['auth_url'].toString(),
            "aws_access_key_id": result['user_name'].toString(),
            "aws_secret_access_key": result['password'].toString(),
            "training_data_bucket": result['training_data'].toString(),
            "training_results_bucket": result['training_result'].toString(),
            "robustnesscheck_data_bucket": result['training_data'].toString(),
            "robustnesscheck_results_bucket": result['training_result'].toString(),
            "model_id": this.current_model.model_id,
            "dataset_file": "fashion_mnist.npz",
            "networkdefinition_file": "keras_original_model.json",
            "weights_file": "keras_original_model.hdf5",
            "memory": result['memory'] + "Gb",
            "cpus": Number(result['cpu']),
            "gpus": Number(result['gpu']),
            "training_job_name": "robustnesscheck_" + this.current_model.model_id
          };
          this.http.post(art_function_link, formData, { headers: headers, observe: "response" })
              .map(response => {
                return response.body
              }).subscribe(data => {
                  if (data['model_id'] == undefined){
                    alert("This training job doesn't satisfy the requirements for robustness check.")
                  }else{
                    alert("Your " + name + " Robustness Check job is started. " +
                    "The Training ID is \"" + data['model_id'] + "\"");
                  }
                }, error => {console.log(error)});
              }
          });
  }

  AIFForm(){
    var cpu = 1;
    var gpu = 0;
    var memory = 4;
    var learner = 1;
    var name = this.current_model.name
    var aif_function_link = "https://openwhisk.ng.bluemix.net/api/v1/web/ckadner_org_dev/default/fairness_check.json"
      const dialogRef = this.dialog.open(AIFDialog, {
        height: '600px',
        data: { name: name, cpu: cpu, gpu: gpu,
                memory: memory, learner: learner,
                auth_url: "https://s3-api.us-geo.objectstorage.softlayer.net"
              }
      });

      dialogRef.afterClosed().subscribe(result => {
        if (result == undefined){
        }else{
          let header_json = {"Content-Type":"application/json", "X-Require-Whisk-Auth":"fiddle"};
          let headers = new HttpHeaders(header_json);
          var formData = {
            "ffdl_service_url": this.loginData['environment'],
            "basic_authtoken": "test",
            "watson_auth_token": "bluemix-instance-id=test-user",
            "aws_endpoint_url": result['auth_url'].toString(),
            "aws_access_key_id": result['user_name'].toString(),
            "aws_secret_access_key": result['password'].toString(),
            "training_data_bucket": result['training_data'].toString(),
            "training_results_bucket": result['training_result'].toString(),
            "fairnesscheck_data_bucket": result['training_data'].toString(),
            "fairnesscheck_results_bucket": result['training_result'].toString(),
            "model_id": this.current_model.model_id,
            "memory": result['memory'] + "Gb",
            "cpus": Number(result['cpu']),
            "gpus": Number(result['gpu']),
            "training_job_name": "fairnesscheck_" + this.current_model.model_id
          };
          this.http.post(aif_function_link, formData, { headers: headers, observe: "response" })
              .map(response => {
                return response.body
              }).subscribe(data => {
                  if (data['model_id'] == undefined){
                    alert("This training job doesn't satisfy the requirements for fairness check.")
                  }else{
                    alert("Your " + name + " Fairness Check job is started. " +
                    "The Training ID is \"" + data['model_id'] + "\"");
                  }
                }, error => {console.log(error)});
              }
          });
  }

deployForm(){
    var deploy_function_link = "http://deployment-python.default.aisphere.info"
      const dialogRef = this.dialog.open(DeployDialog, {
        height: '600px',
        data: { auth_url: "https://s3-api.us-geo.objectstorage.softlayer.net",
                training_result: training_result,
                user_name: user_name,
                password: password
              }
      });

      dialogRef.afterClosed().subscribe(result => {
        if (result == undefined){
        }else{
          let header_json = {"Content-Type":"application/json", "Host" : "deployment-python.default.example.com"};
          let headers = new HttpHeaders(header_json);
          var frameworkname = "tomcli/seldon-core-s2i-python3:0.4";
          var model_file_name = "keras_original_model.hdf5";
          if(this.current_model.framework.name == "pytorch"){
              frameworkname = "tomcli/seldon-gender:0.5";
              model_file_name = "model.pt"
          }
          var formData = {
            "public_ip": "169.48.165.244",
            "aws_endpoint_url": result['auth_url'].toString(),
            "aws_access_key_id": result['user_name'].toString(),
            "aws_secret_access_key": result['password'].toString(),
            "training_results_bucket": result['training_result'].toString(),
            // "framework": result['framework'].toString(),
            "model_file_name": model_file_name,
            "deployment_name": this.current_model.model_id,
            "training_id": this.current_model.model_id,
            "container_image": frameworkname,
            "check_status_only": false
          };
          var model_link = "http://169.48.165.244:30559/seldon/" + this.current_model.model_id + "/api/v0.1/predictions"
          this.http.post(deploy_function_link, formData, { headers: headers, observe: "response" })
              .map(response => {
                return response.body
              }).subscribe(data => {
                  console.log(data);
                  if (data['deployment_status'] == undefined){
                    alert("This training job isn't satisfy for deployment.")
                  }else{
                    alert("Your " + name + " deployment is started. " +
                    "Please go to the kubernetes cluster to check for your deployment. Your Model will be served at \"" + data["deployment_url"] + "\"");
                  }
                }, error => {console.log(error)});
              }
          });
  }

  onSubmit() {
    this.loading = true;
    this.formData = new FormData();
    this.createForm();
    var builtForm = true;
    try{
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
    } catch (error) {
      alert("Model definition zip or manifest file is incorrect or not uploaded!")
      return
    }
    var isConfirmed = confirm("Create training job?");
    if (isConfirmed) {
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
      if(this.on_pipeline_page == true){
        if(this.art_current_model != undefined && this.art_current_model.training.training_status.status != 'COMPLETED'){
          this.artUpdate(this.trainingId);
        }
        //this.deploymentUpdate(this.trainingId);
      }
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

  delete_deployment(id: String) {
    var isConfirmed = confirm("Are you sure you want to delete " + id + " deployment?");
    if (isConfirmed) {
      this.notificationService.info('Deleting deployment', 'ID: ' + id);
      let header_json = {"Content-Type":"application/json"};
      let headers = new HttpHeaders(header_json);
      var formData = {
        "public_ip": "169.48.165.244",
        "deployment_name": id.toString(),
        "training_id": id.toString(),
        "delete_deployment": "true"
      };
      this.http.delete("http://deployment-python.default.aisphere.info", { headers: headers, params: formData }).subscribe(
        data => {
          this.notificationService.success('Deployment deleted.', 'ID: ' + id);
          console.log(data)
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

@Component({
  selector: 'art-dialog',
  templateUrl: './art-dialog.html',
})
export class ArtDialog {
  constructor(
    public dialogRef: MatDialogRef<ArtDialog>,
    @Inject(MAT_DIALOG_DATA) public data: any) { }

  cpu = new FormControl('', [Validators.required, Validators.min(0)]);

  onNoClick(): void {
    this.dialogRef.close();
  }

  getErrorMessage() {
    return this.cpu.hasError('required') ? 'You must enter a value' :
        this.cpu.hasError('min') ? 'Not a valid number' :
            '';
  }
}

@Component({
  selector: 'deploy-dialog',
  templateUrl: './deploy-dialog.html',
})
export class DeployDialog {
  constructor(
    public dialogRef: MatDialogRef<DeployDialog>,
    @Inject(MAT_DIALOG_DATA) public data: any) { }

  onNoClick(): void {
    this.dialogRef.close();
  }
}

@Component({
  selector: 'aif-dialog',
  templateUrl: './aif-dialog.html',
})
export class AIFDialog {
  constructor(
    public dialogRef: MatDialogRef<AIFDialog>,
    @Inject(MAT_DIALOG_DATA) public data: any) { }

  onNoClick(): void {
    this.dialogRef.close();
  }
}
