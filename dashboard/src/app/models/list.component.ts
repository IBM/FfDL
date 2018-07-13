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
