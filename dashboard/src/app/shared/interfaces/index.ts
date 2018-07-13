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
