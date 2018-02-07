import {Component, OnInit, ViewEncapsulation} from '@angular/core';
import * as chartjs from 'chart.js'
import * as $ from 'jquery';
import {DlaasService} from '../shared/services';

@Component({
  selector: 'analytics-main',
  templateUrl: './main.component.html',
  encapsulation: ViewEncapsulation.None
})
export class AnalyticsMainComponent implements OnInit {

  charts: any[] = [];
  time_from: any = 0;
  time_to: any = 0;

  constructor(private dlaas: DlaasService) {
  }

  ngOnInit() {
    this.show('dailyJobs');
    this.loadCharts();
  }

  // TODO: make this more efficient by either
  // (1) filtering server-side, or
  // (2) not reloading the data when filtering client-side
  filterDates(entries: Map<string, any>[]): Map<string, any>[] {
    let result: Map<string, any>[] = [];
    this.time_from = parseInt(this.time_from);
    this.time_to = parseInt(this.time_to);
    if(this.time_from < 2000000000) {
      this.time_from *= 1000;
    }
    if(this.time_to < 2000000000) {
      this.time_to *= 1000;
    }

    var self = this;
    entries.forEach(function(entry) {
      var date = Date.parse(entry['training_status']['submission_timestamp']);
      if((self.time_from <= 0 || date >= self.time_from) &&
          (self.time_to <= 0 || date <= self.time_to)) {
        result.push(entry);
      }
    });
    return result;
  }

  mergeEntries(data: Map<string, any>): Map<string, any>[] {
    let result: Map<string, any>[] = [];
    for (var key in data) {
      Array.prototype.push.apply(result, data[key]);
    }
    result = this.filterDates(result);
    return result;
  }

  extractMap(data: any[], keyFunction: (name: any) => string) {
    var result = {};
    data.forEach(function (entry) {
      var key = keyFunction(entry);
      if (!result[key]) {
        result[key] = [];
      }
      result[key].push(entry);
    });
    return result;
  }

  extractDays(data: any) {
    return this.extractMap(data, function (entry) {
      return entry.training_status.submission_timestamp.substring(0, 10);
    });
  }

  extractCPUs(data: any) {
    return this.extractMap(data, function (entry) {
      return entry.training.resources.cpus;
    });
  }

  extractGPUs(data: any) {
    return this.extractMap(data, function (entry) {
      return entry.training.resources.gpus;
    });
  }

  extractGPUsSum(data: any) {
    return this.extractMap(data, function (entry) {
      var learners = entry.training.resources.learners || 1;
      return (entry.training.resources.gpus * learners).toString();
    });
  }

  extractFrameworks(data: any) {
    return this.extractMap(data, function (entry) {
      return entry.model_definition.framework.name;
    });
  }

  extractLearners(data: any) {
    return this.extractMap(data, function (entry) {
      return entry.training.resources.learners;
    });
  }

  chartDailyUsers(data: any) {
    var entries = this.mergeEntries(data['results']);
    var daysMap = this.extractDays(entries);
    var days = Object.keys(daysMap).sort();
    var list: any[] = [];
    days.forEach(function (day) {
      var users: string[] = [];
      daysMap[day].forEach(function (entry: any) {
        if (users.indexOf(entry.user_id) < 0) {
          users.push(entry.user_id);
        }
      });
      list.push(users.length);
    });
    this.renderChart('chart_dailyUsers', 'Daily Users', list, days);
  }

  chartDailyJobs(data: Map<string,any>) {
    var entries = this.mergeEntries(data["results"]);
    var daysMap = this.extractDays(entries);
    var days = Object.keys(daysMap).sort();
    var values: number[] = [];
    days.forEach(function (day) {
      values.push(daysMap[day].length);
    });
    this.renderChart('chart_dailyJobs', 'Daily Jobs', values, days);
  }

  chartGPUsTimeline(data: any) {
    var entries = this.mergeEntries(data.results);
    var daysMap = this.extractDays(entries);
    var days = Object.keys(daysMap).sort();
    var values: number[] = [];
    days.forEach(function (day) {
      var gpusSum = 0;
      daysMap[day].forEach(function (entry: any) {
        gpusSum += entry.training.resources.gpus;
      });
      values.push(gpusSum);
    });
    this.renderChart('chart_GPUs_timeline', 'Daily GPU Usage', values, days);
  }

  chartCPUs(data: any) {
    var entries = this.mergeEntries(data.results);
    var cpusMap = this.extractCPUs(entries);
    var cpus = Object.keys(cpusMap);
    var values: number[] = [];
    cpus.forEach(function (cpu) {
      values.push(cpusMap[cpu].length);
    });
    this.renderChart('chart_CPUs', 'Job CPUs', values, cpus, 'pie');
  }

  chartGPUs(data: any) {
    var entries = this.mergeEntries(data.results);
    var gpusMap = this.extractGPUs(entries);
    var gpus = Object.keys(gpusMap);
    var values: number[] = [];
    gpus.forEach(function (gpu) {
      values.push(gpusMap[gpu].length);
    });
    this.renderChart('chart_GPUs', 'GPUs per Trainer', values, gpus, 'pie');
  }

  chartGPUsSum(data: any) {
    var entries = this.mergeEntries(data.results);
    var gpusMap = this.extractGPUsSum(entries);
    var gpus = Object.keys(gpusMap);
    var values: number[] = [];
    gpus.forEach(function (gpu) {
      values.push(gpusMap[gpu].length);
    });
    this.renderChart('chart_sumGPUs', 'GPUs per Job', values, gpus, 'pie');
  }

  chartFrameworks(data: any) {
    var entries = this.mergeEntries(data.results);
    var map = this.extractFrameworks(entries);
    var keys = Object.keys(map);
    var values: number[] = [];
    keys.forEach(function (key) {
      values.push(map[key].length);
    });
    this.renderChart('chart_frameworks', 'DL Frameworks', values, keys, 'pie');
  }

  chartLearners(data: any) {
    var entries = this.mergeEntries(data.results);
    var map = this.extractLearners(entries);
    var keys = Object.keys(map);
    var values: number[] = [];
    keys.forEach(function (key) {
      values.push(map[key].length);
    });
    this.renderChart('chart_learners', '# Learners', values, keys, 'pie');
  }

  chartQueueingTimes(data: any) {
    var buckets = [4, 6, 8, 10, 12, 14, 16, -1];
    return this.chartTimes(data, 'queueTimes', 'Queueing Times', buckets,
      function(entry) {
        return Date.parse(entry.training_status.submission_timestamp);
      }, function(entry) {
        return Date.parse(entry.training_status.download_start_timestamp);
      });
  }

  chartDownloadTimes(data: any) {
    var buckets = [2, 4, 8, 16, 32, 64, 128, -1];
    return this.chartTimes(data, 'dlTimes', 'Downloading Times', buckets,
      function(entry) {
        return Date.parse(entry.training_status.download_start_timestamp);
      }, function(entry) {
        return Date.parse(entry.training_status.process_start_timestamp);
      });
  }

  chartTrainTimes(data: any) {
    return this.chartTimes(data, 'trainTimes', 'Training Times', null,
      function(entry) {
      return Date.parse(entry.training_status.process_start_timestamp);
    }, function(entry) {
      return Date.parse(entry.training_status.completion_timestamp);
    });
  }

  chartTimes(data: any, chartID: string, title: string, buckets: any[],
             lower: (entry: any) => number, upper: (entry: any) => number) {
    var entries = this.mergeEntries(data.results);
    if(!buckets) {
      buckets = [15, 30, 60, 120, 240, 480, 960, -1];
    }
    var values: number[] = [];
    while(values.length < buckets.length) {
      values.push(0);
    }
    entries.forEach(function (entry) {
      var start = lower(entry);
      var end = upper(entry);
      var duration = end - start;
      if(isNaN(duration) || duration <= 0) {
        return;
      }
      for(var i = 0; i < buckets.length; i ++) {
        if(buckets[i] < 0 || duration < buckets[i] * 1000) {
          values[i] ++;
          break;
        }
      }
    });
    for(var i = 0; i < buckets.length; i ++) {
      buckets[i] = '< ' + buckets[i] + 's';
    }
    buckets[buckets.length - 1] = 'max';
    this.renderChart('chart_' + chartID, title, values, buckets, 'bar');
  }

  isSourceSelected(source: string) {
    return $('#source_' + source).prop('checked');
  }

  ajax(request: any) {
    return $.ajax(request);
  }

  loadData() {
    var sources: any[] = [];
    var self = this;
    ['local', 'cruiser1', 'cruiser2'].forEach(function (source) {
      if (self.isSourceSelected(source)) {
        sources.push(source);
      }
    });
    return this.dlaas.getTrainingJobs(sources);
  }

  resetCharts() {
    this.charts.forEach(function (chart: any) {
      chart.destroy();
    });
    this.charts = [];
  }

  renderChart(id: string, title: string, data: any[], ticks: any[], type: string = null) {
    var ctx = ($('#' + id)[0] as HTMLCanvasElement).getContext('2d');
    var datasets = [];
    var entry = {
      label: title || 'Chart',
      data: data,
      backgroundColor: bgColors
    };
    if (type == 'pie') {
      var bgColors: any[] = [];
      ticks.forEach(function () {
        var color = 'rgb(' + Math.round(Math.random() * 200) + ',' +
          Math.round(Math.random() * 200) + ',' +
          Math.round(Math.random() * 200) + ')';
        bgColors.push(color);
      });
      entry.backgroundColor = bgColors;
    }
    datasets.push(entry);
    type = type || 'line';
    var chart = new chartjs.Chart(ctx, {
      type: type,
      data: {
        labels: ticks,
        datasets: datasets
      }
    });
    this.charts.push(chart);
    $('#loading').hide();
  }

  show(id: string) {
    $('.content').hide();
    $('#div_' + id).show();
    $('#div_' + id + ' .content').show();
  }

  loadCharts() {
    $('#loading').show();
    $('#errorMsg').text("");
    this.resetCharts();
    let self = this;
    this.loadData().subscribe(function (data: any) {
      var dataMap: Map<string,any> = data
      self.chartDailyJobs(dataMap);
      self.chartDailyUsers(dataMap);
      self.chartCPUs(dataMap);
      self.chartGPUs(dataMap);
      self.chartGPUsSum(dataMap);
      self.chartGPUsTimeline(dataMap);
      self.chartFrameworks(dataMap);
      self.chartLearners(dataMap);
      self.chartQueueingTimes(dataMap);
      self.chartDownloadTimes(dataMap);
      self.chartTrainTimes(dataMap);
    }, function (error) {
      $('#errorMsg').text("Error loading the data.");
      $('#loading').hide();
    });
  }

}
