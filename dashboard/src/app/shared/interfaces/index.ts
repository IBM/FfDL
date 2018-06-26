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

// Defines a connection to a DLaaS service
export interface DlaasConnection {
  id: string;
  url: string;
  username: string;
  password: string;
}

export interface ModelMetadata {
  id: string;
  name: string;
  version: string;
  framework: string;
  description: string;
}

export interface TrainingData {
  id: string;
  description: string;
  status: string;
  model_id: string;
}

export interface Question {
  id?: string;
  title?: string;
  text?: string;
}
