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

export class LoginData {
  environment: string;
  token: string;
  username: string;
  expiration: string;
  role: string;

  constructor(environment: string, username: string, token: string, expiration: string, role: string) {
    this.environment = environment;
    this.username = username;
    this.token = token;
    this.expiration = expiration;
    this.role = role;
  }

  toString(): string {
    return this.environment + ' ' + this.username + ' ' + this.token + ' ' + this.expiration;
  }
}

export interface BasicNewModel {
  model_id: string;

  location: string;
}

export interface ModelData {
  model_id: string;
  name: string;
  description: string;
  training: Training;
  framework: Framework;
  data_stores: DataStore[];
}

export interface Training {
  command: string;
  cpus: number;
  gpus: number;
  memory: number;
  memory_unit: string;
  learners: number;
  training_status: TrainingStatus;
}

export interface LogLine {
  meta: MetaInfo;
  line: string;
}

export enum AnyDataType {
  STRING = 0,
  JSONSTRING = 1,
  INT = 2,
  FLOAT = 3,
}

export interface TypedAny {
    type: string;
    value: string;
}

export interface KeyValue {
  key: string;
  value: TypedAny;
}

export interface EMetrics {
  meta: MetaInfo;
  etimes: { [key: string]: TypedAny };
  grouplabel: string;
  values: { [key: string]: TypedAny };
}

export interface Framework {
  name: string;
  version: string;
}

export interface DataStore {
  data_store_id: string;
  type: string;
  connection: Object;
}

export interface MetaInfo {
  training_id: string;
  user_id: string;
  time: string;
  rindex: string;
}


export interface TrainingStatus {
  status: string;
  status_description: string;
  submitted: string;
  completed: string;
}
