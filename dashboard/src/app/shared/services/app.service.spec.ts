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

import {} from 'jasmine';
import { inject, async, ComponentFixture, TestBed } from '@angular/core/testing';
import { By }           from '@angular/platform-browser';
import { DebugElement } from '@angular/core';
import { Title } from '@angular/platform-browser';

import { AppService } from './app.service';

describe('AppService', function() {

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ Title,  AppService ]
    })
      .compileComponents();
  }));

  it('setTitle() works', inject([AppService], (appService: AppService) => {
    let newTitle = '';
    appService.setTitle(newTitle);
    expect(appService.getTitle()).toEqual(`${appService.baseTitle}`);
    newTitle = 'abc';
    appService.setTitle(newTitle);
    expect(appService.getTitle()).toEqual(`${newTitle}${appService.titleSeparator}${appService.baseTitle}`);
  }));

  it('inferTitleFromUrl() works', inject([AppService], (appService: AppService) => {
    const newTitle = 'Question Add';
    appService.inferTitleFromUrl('question/add');
    expect(appService.getTitle()).toEqual(`${newTitle}${appService.titleSeparator}${appService.baseTitle}`);
  }));

});
