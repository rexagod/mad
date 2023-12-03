# Metrics Anomaly Detector (MAD)

## Description

The Metrics Anomaly Detector, abbreviated as MAD, is a Kubernetes controller that monitors the health of a set of components.

## Usage

After installation, the controller starts monitoring the health of the specified endpoints. The *aggregated* health check events are stored in a [circular buffer](https://pkg.go.dev/container/ring).

```yaml
apiVersion: mad.instrumentation.k8s-sigs.io/v1alpha1
kind: MetricsAnomalyDetectorResource
metadata:
  name: metrics-anomaly-detector-resource
  namespace: default
spec:
  # bufferSize is the size of the buffer that stores health check events.
  # This is updated at every `queryInterval` seconds.
  # A buffer value consists of the following:
  # * healthy: The aggregate health status of all the specified endpoints. This is false if any of the endpoints are unhealthy.
  # * timestamp: The timestamp of the health check event.
  # The `status.CurrentBufferSize` denotes the current size of the buffer.
  # The `status.LastBufferModificationTime` denotes the timestamp of the last buffer modification.
  # The `status.LastBuffer` denotes the last buffer snapshot. This comes in handy between the controller restarts, so that the buffer is not lost.
  bufferSize: 200 # 200 is the default value.
  # healthcheckEndpoints is a list of endpoints to monitor.
  # The `status.HealthcheckEndpointsHealthy` denotes the health status of each individual endpoint. The value is false if the endpoint is unhealthy, including the case where a connection was not established.
  # The `status.LastHealthcheckQueryTime` denotes the timestamp of the last health check query.
  healthcheckEndpoints:
    - "https://kubernetes.default/readyz"
    - "https://foo-service.baz-namespace.svc.cluster.local/healthz"
    - "https://bar-service.baz-namespace.svc.cluster.local/readyz"
  # queryInterval is the interval at which the endpoints are queried, in seconds.
  queryInterval: 60 # 60 is the default value.
```

The size of the buffer, the endpoints to monitor, and the query interval can be configured through a `MetricsAnomalyDetectorResource` CR. Multiple instances of the CR can be created to monitor different sets of endpoints, or isolate operational logic for better maintainability.

## WIP

- [ ] Add support for streaming RPC queries. These should:
  - [ ] Accept time-intervals defining a pair of windows.
  - [ ] Compare the state of the monitored scope within the two windows, using [isolation-forests](https://ars.els-cdn.com/content/image/1-s2.0-S0952197622004936-fx1_lrg.jpg).
  - [ ] Return the anomalies detected within the two windows.
- [ ] Deploy and reconcile manifests using the controller.
- [ ] Add e2e tests.

## License

This project is licensed under the Apache License, Version 2.0. You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.
