# Volume Controller for Kubernetes Integration

### Current usage
1. Set up the intel VCK and download your S3 data to the hostpath for all your nodes.
   * https://github.com/IntelAI/vck/blob/master/docs/user.md
2. Run the training job with host_mount mode
   * example manifest.yml
   ``` yaml
   name: tf_convolutional_network_tutorial
   description: Convolutional network model using tensorflow
   version: "1.0"
   gpus: 0
   cpus: 0.25
   memory: 1Gb

   # Object stores that allow the system to retrieve training data.
   data_stores:
     - id: hostmount
       type: mount_volume
       training_data:
         container: vck-resource-f0e5a3ba-1744-11e8-a808-0a580a44065b
       training_results:
         container: vck-resource-f0e5a3ba-1744-11e8-a808-0a580a44065b
       connection:
         type: host_mount
         name: "host-mount"
         path: "/var/datasets/"

   framework:
     name: tensorflow
     version: "1.5.0-py3"
     command: >
       python3 convolutional_network.py --trainImagesFile ${DATA_DIR}/train-images-idx3-ubyte.gz
         --trainLabelsFile ${DATA_DIR}/train-labels-idx1-ubyte.gz --testImagesFile ${DATA_DIR}/t10k-images-idx3-ubyte.gz
         --testLabelsFile ${DATA_DIR}/t10k-labels-idx1-ubyte.gz --learningRate 0.001
         --trainingIters 20000

   evaluation_metrics:
     type: tensorboard
     in: "$JOB_STATE_DIR/logs/tb"
   ```
   
## FfDL Codebase Integration Proposal
1. Implement a new module that handles creating the `volumemanage` for VCK
2. Insert logic to provision `volumemanage` resource and monitor it for completion before executing the training job workload.
3. To make it more elastic, we need to come up with some algorithm on how much data replicas we need for each job. Then create some labels/tags to allow users to reuse the same dataset volume.
4. Need to figure out a shared file storage for all the learner pods (required for many distributed learning methods) and a way to store the model results for our users.
