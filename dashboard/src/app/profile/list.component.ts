import { Component, ChangeDetectionStrategy, ViewEncapsulation, OnInit } from '@angular/core';
// import { AsyncPipe } from '@angular/common';
// import { ROUTER_DIRECTIVES } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { DlaasService } from '../shared/services';
import { DlaasConnection } from '../shared/interfaces';

@Component({
    selector: 'my-profile-list',
    templateUrl: './list.component.html',
    // changeDetection: ChangeDetectionStrategy.OnPush,
    // encapsulation: ViewEncapsulation.None,
    // pipes: [AsyncPipe],
    // directives: [ROUTER_DIRECTIVES]
})
export class ProfileListComponent implements OnInit {

    connections: Observable<DlaasConnection[]>;

    constructor(private dlaas: DlaasService) { }

    ngOnInit() {
        this.find();
    }

    find() {
        // this.connections = this.dlaas.getConnections();
    }


}
