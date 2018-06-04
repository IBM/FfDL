# Perform Automated Machine Learning With H2O on FfDL

[H2O.ai](https://h2o.ai) provides an open source platform for automated machine learning: [H2O-3](https://www.h2o.ai/h2o/)

H2O is an open source, in-memory, distributed, fast, and scalable machine learning and predictive analytics platform that allows you to build machine learning models on big data and provides easy productionalization of those models in an enterprise environment.

# Deployment Steps

1. Follow steps to deploy FfDL from the [user guide](https://github.com/IBM/FfDL/blob/master/docs/user-guide.md)
2. Add some data, either follow the user guide to store the data locally or host the data in a cloud storage bucket and pull it at runtime.
3. Change the manifest.yaml to the settings that you want
4. Once FfDL is deployed in your Kubernetes cluster, use the CLI or GUI to deploy H2O

# Examples
sample deployment scripts are hosted under: FfDL/community/FfDL-H2Oai

If you need a sample dataset, you can pull this toy dataset:
Train Set:
s3://h2o-public-test-data/smalldata/higgs/higgs_train_10k.csv
Test Set:
s3://h2o-public-test-data/smalldata/higgs/higgs_test_5k.csv
