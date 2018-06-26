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

import {Component, Input, OnInit, OnDestroy, OnChanges, ViewEncapsulation} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import { Subscription } from 'rxjs/Subscription';
import {DlaasService} from '../shared/services';
import {EMetrics, KeyValue, TypedAny} from "../shared/models/index";
import * as chartjs from 'chart.js'
import {ChartPoint, ChartDataSets, ChartData, ChartColor} from 'chart.js'
import {Chart as Chart} from 'chart.js'
import * as $ from 'jquery';

@Component({
  selector: 'training-emetrics',
  templateUrl: './emetrics.component.html',
  styleUrls: ['./emetrics.component.css'],
  encapsulation: ViewEncapsulation.None
})
export class TrainingEMetricsComponent implements OnInit, OnChanges {

  @Input() private trainingId: string;
  // @Input() private dataStream: Observable<any>; // TODO add type

  /** Dictionary of scalar key to index that corresponds to slot in charts array. */
  public scalarsIndexDict:{[id: string] : number} = {};

  /**
   * This should always point to the first free empty place in charts array.
   * If it is equal or greater that charts.length, room has to be made.
   */
  firstFreeScalarIndex: number = 0;

  /**
   * The number of populated charts on the page, each chart tracks the a
   * scalar value from the evaluation metrics record.
   */
  public charts: Array<Chart> = [];
  public groupCategoryColors: ChartColor = [
    'MediumBlue',
    'DarkOrange',
    'DarkGreen',
    'Indigo',
    'DarkRed',
    'Sienna'
  ];

  public numberPagesPerUpdate: number = 1;

  private subscription: Subscription;

  private showSpinner = false;

  private pos: number = 0;
  private pagesize: number = 10;
  private prevTime: string = "";

  private findSub: Subscription;

  public follow: boolean = false;

  constructor(private dlaas: DlaasService) {
  }

  ngOnChanges(changes: any) {
  }

  sleep(ms = 0) {
    return new Promise(r => setTimeout(r, ms));
  }

  startOngoingUpdate() {
    this.subscription = Observable.interval(1000*4).subscribe(x => {
      this.find(this.pos, 100, "");
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

  ngOnInit() {
    for (let i = 0; i < 1; i++) {
      this.find(this.pos, this.pagesize, "");
    }
    this.showSpinner = false;
  }

  ngOnDestroy() {
    if (this.subscription) {
      this.subscription.unsubscribe();
      this.subscription = null
    }
  }

  /** Reset to empty all charts */
  resetCharts() {
    this.charts.forEach(function (chart: any) {
      chart.destroy();
    });
    this.charts = [];
    this.firstFreeScalarIndex = 0;
  }


  private getChartIndex(k: string): number {
    if (!(k in this.scalarsIndexDict)) {
      this.scalarsIndexDict[k] = this.firstFreeScalarIndex;
      this.firstFreeScalarIndex++
    }
    return this.scalarsIndexDict[k]
  }


  /** Create new chart */
  makeNewLineChart(id: string, title: string,
                   datasets: Array<ChartDataSets>,
                   labels: Array<string | string[]>): Chart {
    let ctx = ($('#' + id)[0] as HTMLCanvasElement).getContext('2d');

    let type = 'scatter';
    let chart = new chartjs.Chart(ctx, {
      type: type,
      data: {
        labels: labels,
        datasets: datasets
      },
      options: {
        title: {
          display: true,
          text: title
        },
        showLines: true,
        spanGaps: true,
        maintainAspectRatio: true,
        elements: {
          point: {
            backgroundColor: 'transparent',
            radius: 0,
          },
          line: {
            backgroundColor: 'transparent',
            borderWidth: 1,
            tension: 0,
          },
        },
        scales: {
          xAxes: [{
            type: 'linear',
            position: 'bottom',
            stacked: true
          }]
        }
      }
    });
    return chart;
  }


  updateLinechartWithPointValue(chart: Chart, chartPoint: ChartPoint, groupLabel: string, temporalLabel: string ) {
    let dataSets: Array<ChartDataSets> = chart.data.datasets;
    let groupChartDataSet: ChartDataSets = null;
    let longestSpan: number = 0;
    for ( let whichDataSet of dataSets ){
      if ( whichDataSet.label == groupLabel ) {
        groupChartDataSet = whichDataSet;
        if( groupChartDataSet.data.length >  longestSpan) {
          longestSpan = groupChartDataSet.data.length;
        }
      }
    }
    if ( groupChartDataSet == null ) {
      groupChartDataSet = {
          data: [],
          label: groupLabel,
          borderColor: this.groupCategoryColors[chart.data.datasets.length],
          spanGaps: true,
        };
      chart.data.datasets = [...chart.data.datasets, groupChartDataSet];
    }

    groupChartDataSet.data = <ChartPoint[]>[...groupChartDataSet.data, chartPoint];

    let lastIndex: number = chart.data.labels.length-1;
    if (chart.data.labels[lastIndex] != temporalLabel) {
      chart.data.labels = [...chart.data.labels, temporalLabel]
    }
  }


  getChartOrMakeRoom(scalerIndex: number): Chart {
    if ( scalerIndex >= this.charts.length ) {
      // Warning, not certain how supported this is.
      let originalLength = this.charts.length;
      this.charts.length = scalerIndex+1; // Might not be needed
      this.charts = this.charts.fill(null, originalLength, scalerIndex)
    }

    return this.charts[scalerIndex];
  }


  show(id: string) {
    $('.content').hide();
    $('#div_' + id).show();
    $('#div_' + id + ' .content').show();
  }


  fetchTemporalValueMaybe(evaluationMetricsRecord: EMetrics, eKey: string): {val: number, label: string} {
    let valLabelPair: {val: number, label: string} = null;
    if (eKey in evaluationMetricsRecord.etimes) {
      let typedVal: TypedAny = evaluationMetricsRecord.etimes[eKey];
      if (typedVal && (typedVal.type == "INT" || typedVal.type == "FLOAT")) {
        valLabelPair = {val: Number(typedVal.value), label: typedVal.value}
      }
    }
    return valLabelPair;
  }


  getLikelyTemporalValue(evaluationMetricsRecord: EMetrics): {val: number, label: string} {
    // Search through list in increasing orders of preference
    // Hacksville?  Or no?
    for ( let etimeKey of ["iteration", "epoch", "step", "tick", "time"] ) {
      let valLabelPair: {val: number, label: string} =
        this.fetchTemporalValueMaybe(evaluationMetricsRecord, etimeKey);
      if ( valLabelPair != null) {
        return valLabelPair;
      }
    }
    // hmmm, anything typed as a number?
    for (let eKey in evaluationMetricsRecord.etimes) {
      let typedVal: TypedAny = evaluationMetricsRecord.etimes[eKey];
      if (typedVal && (typedVal.type == "INT" || typedVal.type == "FLOAT")) {
        return {val: Number(typedVal.value), label: typedVal.value}
      }
    }
    // Fall back to the meta time, which may or may not be reliable.
    return {val: Number(evaluationMetricsRecord.meta.time), label: <string>(evaluationMetricsRecord.meta.time)}
  }

  private find(pos: number, pagesize: number, since: string) {
    $('#errorMsg').text("");
    this.findSub = this.dlaas.getTrainingMetrics(this.trainingId, pos, pagesize, since).subscribe(
      data => {
        let eMetricsList: Array<EMetrics> = data;
        if (eMetricsList.length == 0) {
          return;
        }
        let lastIndex = eMetricsList.length - 1;
        this.prevTime = eMetricsList[lastIndex].meta.time;

        for(let eMetricsListIndex: number = 0; eMetricsListIndex < eMetricsList.length; eMetricsListIndex++) {
          let evaluationMetricsRecord: EMetrics  = eMetricsList[eMetricsListIndex];

          for (let scalerKey in  evaluationMetricsRecord.values) {
            let scalerChartIndex: number = this.getChartIndex(scalerKey);

            let typedVal: TypedAny = evaluationMetricsRecord.values[scalerKey];

            if (!(typedVal && (typedVal.type == "INT" || typedVal.type == "FLOAT"))) {
              continue
            }
            let chart = this.getChartOrMakeRoom(scalerChartIndex);
            // chart may be null at this point, if the chart hasn't been made.  The slot
            // in the charts array, however is guaranteed to exist.

            let valueAndLabel: {val: number, label: string} = this.getLikelyTemporalValue(evaluationMetricsRecord);
            let chartPoint: ChartPoint = {
              x: valueAndLabel.val,
              y: Number(typedVal.value),
            };
            if ( chartPoint.x == 0 ) {
              continue
            }

            if ( chart == null ) {
              let componentId: string = 'chart_scalar_'+scalerChartIndex;
              let dataSets: Array<ChartDataSets> = [
                {
                  data: [chartPoint],
                  label: evaluationMetricsRecord.grouplabel,
                  borderColor: this.groupCategoryColors[0],
                  spanGaps: true,
                }
              ];
              let labels: Array<string> = [valueAndLabel.label];
              chart = this.makeNewLineChart(componentId, scalerKey, dataSets, labels);
              this.charts[scalerChartIndex] = chart;
            } else {
              this.updateLinechartWithPointValue(chart, chartPoint,
                evaluationMetricsRecord.grouplabel, valueAndLabel.label )
            }
          }
        }
        this.pos += lastIndex;
        $('#loading').hide();
        this.updateAllCharts()
       },
      err => {
        // $('#errorMsg').text("Error loading the data.");
        this.follow = false;
        this.stopOngoingUpdate();

        $('#loading').hide();
      }
    );
  }


  public updateAllCharts():void {
    for (let scalerIndex: number = 0; scalerIndex < this.charts.length; scalerIndex++) {
      let lineChart: Chart = this.charts[scalerIndex];
      if ( lineChart ) {

        // The lines below are forcing a live update.
        // As far as I can tell, it's the only way to get the thing to update after a
        // data append operation. I'd like to find a real expert in D3 charting to confirm this, and tell if there
        // is a better way, because that reassignment is ridiculous. There is an update method on the chart, but
        // I don't think it works like one would expect.
        // TODO: It's possible that some of those patterns might not be needed? Do another round a experimentation to make certain.
        lineChart.data.datasets = [...lineChart.data.datasets];
        lineChart.data.labels = [...lineChart.data.labels];
        lineChart.update()
      }
    }
  }


  public updateData():void {
    for (var i = 0; i < this.numberPagesPerUpdate; i++) {
      this.find(this.pos, 50, "");
    }

  }

  // events
  public chartClicked(e:any):void {
    console.log(e);
  }


  public chartHovered(e:any):void {
    console.log(e);
  }

}
