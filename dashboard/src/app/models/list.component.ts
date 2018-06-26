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

import { Component, OnInit } from '@angular/core';
// import { AsyncPipe } from '@angular/common';
// import { ROUTER_DIRECTIVES } from '@angular/router';
// import { Observable } from 'rxjs/Observable';
import { DlaasService } from '../shared/services';
import { ModelData } from "../shared/models/index";

@Component({
    selector: 'my-models-list',
    templateUrl: './list.component.html',
    // changeDetection: ChangeDetectionStrategy.OnPush,
    // encapsulation: ViewEncapsulation.None,
    // pipes: [AsyncPipe],
    // directives: [ROUTER_DIRECTIVES]
})
export class ModelsListComponent implements OnInit {

    models: ModelData[];
    modelsError: Boolean = false;

    constructor(private dlaas: DlaasService) { }

    ngOnInit() {
        this.find();
    }

    find() {
        this.dlaas.getTrainings().subscribe(
            data => { this.models = data; },
            err => { this.modelsError = true; }
        );
    }

}
