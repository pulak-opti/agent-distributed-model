# Hack Day Project(Q2)

## Topic: Datafile synchronization across Agent instances

### Background

Currently, eventual consistency is maintained across datafile of Agent instances. 

Details: https://docs.developers.optimizely.com/feature-experimentation/docs/optimizely-agent#datafile-synchronization-across-agent-instances

In this POC, eventual consistency is improved to a significant level using a pub/sub system. In this POC, [NATS](https://nats.io/) is used as the pub/sub system.

### Architecture

![arch](/static/arch.png)

### Workflow
1. When multiple agent instances are running in a distributed system like Kubernetes, user can able to register webhook in app.optimizely.com to get updated datafile notification.

2. In this case, only one agent instance will be notified via the webhook.

3. That instance will publish the updated datafile to NATS(pub/sub system).

4. NATS will immediately send the updated datafile to all the agent instances as they all are subscribed to a specific channel for the updated datafile.

5. Each agent instances have updated datafile within milliseconds.

### Run this POC in Kubernetes

1. Install Kind(Local Kubernetes cluster): https://kind.sigs.k8s.io/docs/user/quick-start/#installation

2. Deploy nats helm chart with jetstream enabled:

    custom values file for enabling the jetstream: 

    ```yaml
    nats:
    jetstream:
        enabled: true

        memStorage:
        enabled: true
        size: 2Gi

        fileStorage:
        enabled: true
        size: 1Gi
        storageDirectory: /data/
    ```

    ```bash
    $ helm repo add nats https://nats-io.github.io/k8s/helm/charts/
    $ helm repo update
    $ helm install my-nats nats/nats -n nats --create-namespace --values values.yaml
    ```

3. Prepare docker image and load in Kind cluster:
   ```
   $ git clone https://github.com/pulak-opti/agent-distributed-model.git
   $ cd agent-distributed-model
   $ make deploy-to-kind
   ```

4. Deploy this POC as Deployment in Kubernetes:
   - create a demo namespace to run the workload components:
     ```bash
     $ kubectl create ns deme
     ```
   - go to deploy directory and adjust the sdk key in secret.yaml
   - create the secret
     ```bash
     $ kubectl apply -f secret.yaml
     ```
   - adjust the image name(if necessary) in the deployment.yaml
   - create the k8s deployment and service
     ```bash
     $ kubectl apply -f deployment.yaml
     ```

5. Mock the webhook call using kubectl service port forwarding:
   ```bash
   $ kubectl port-forward svc/agent-svc -n demo 8080:8080
   # from another terminal
   $ curl localhost:8080/optimizely/webhook
   ```

6. Check pods log:
   ```bash
   $ kubectl logs -n demo <pod-name>
   ```
   You'll find that only one instance has got the webhook api call & published the updated datafile to NATS(pub/sub system). Then, all instances are got updated datafile msg from NATS as subscribed.

