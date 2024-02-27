# [WIP] Metrics Anomaly Detector (MAD)

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
  bufferSize: 10 # 10 is the default value.
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
Below is an example of a populated `MetricsAnomalyDetectorResource` CR.

<details>
<summary>CR</summary>

```yaml
apiVersion: mad.instrumentation.k8s-sigs.io/v1alpha1
kind: MetricsAnomalyDetectorResource
metadata:
  annotations:
    name: metrics-anomaly-detector-resource-sample
    namespace: default
spec:
  bufferSize: 10
  healthcheckEndpoints:
    - https://kubernetes.default/readyz
  queryInterval: 60
status:
  currentBufferSize: 10
  healthcheckEndpointsHealthy:
    https://kubernetes.default/readyz: true
  lastBuffer:
    - healthy: true
      timestamp: "2024-02-27T14:20:45Z"
    - healthy: true
      timestamp: "2024-02-27T14:19:33Z"
    - healthy: true
      timestamp: "2024-02-27T14:18:20Z"
    - healthy: true
      timestamp: "2024-02-27T14:17:08Z"
    - healthy: true
      timestamp: "2024-02-27T14:15:57Z"
    - healthy: true
      timestamp: "2024-02-27T14:14:46Z"
    - healthy: true
      timestamp: "2024-02-27T14:13:35Z"
    - healthy: true
      timestamp: "2024-02-27T14:12:25Z"
    - healthy: true
      timestamp: "2024-02-27T14:11:15Z"
    - healthy: true
      timestamp: "2024-02-27T14:10:05Z"
  lastBufferModificationTime: "2024-02-27T14:11:59Z"
  lastHealthcheckQueryTime: "2024-02-27T14:20:45Z"
```

</details>

The size of the buffer, the endpoints to monitor, and the query interval can be configured through a `MetricsAnomalyDetectorResource` CR. Multiple instances of the CR can be created to monitor different sets of endpoints, or isolate operational logic for better maintainability.

### Querying

`mad`'s `compute_health` endpoint takes in the following query parameters:
* `key`: The key of the `MetricsAnomalyDetectorResource` CR. Should be in the format `namespace/name`.
* `ts_a`: The start timestamp of the time range to query, in [RFC3339](https://datatracker.ietf.org/doc/html/rfc3339#section-5.8) format.
* `ts_b`: The end timestamp of the time range to query, in [RFC3339](https://datatracker.ietf.org/doc/html/rfc3339#section-5.8) format.

<details>
<summary>Querying</summary>

```console
┌[rexagod@nebuchadnezzar] [/dev/ttys003]
└[~]> curl "http://localhost:8080/compute_health?key=default/metrics-anomaly-detector-resource-sample&ts_a=2022-01-01T00:00:00Z&ts_b=2024-12-31T23:59:59Z"
{"health_score": 1.000000, "unhealthy_records": 0, "healthy_records": 10}%
```

</details>

## WIP

- [ ] Deploy and reconcile manifests using the controller.
- [ ] Use [isolation-forests](https://ars.els-cdn.com/content/image/1-s2.0-S0952197622004936-fx1_lrg.jpg) as the underlying algorithm.

## License

This project is licensed under the Apache License, Version 2.0. You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.
