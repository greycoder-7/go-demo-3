# Go Demo 3 - Using Horinzontal Pod Autoscaler with Custom Metrics

## Features

- Horizontal Pod Autoscaler using Prometheus custom metrics 

# Table of Contents
1. [Deploy the application example](#deploy-the-application-example)
2. [Prometheus and Grafana deployment](#prometheus-and-grafana-deployment)
3. [Annotations for scraping application metrics](#annotations-for-scraping-application-metrics)
4. [Kubernetes HPA metrics](#kubernetes-hpa-metrics)

## Deploy the application example

### Deploy the application

```sh
kubectl apply -f k8s/configmap.yaml

kubectl apply -f k8s/deploy.yaml

kubectl apply -f k8s/service.yaml
```

### Deploy database

```sh
kubectl apply -f k8s/mongodb.yaml
```

### Create HPA object using custom metrics

```sh
kubectl apply -f k8s/hpa_custom_metrics.yaml
```

## Prometheus and Grafana deployment

### Deploy Prometheus Server
```sh
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

helm install prometheus prometheus-community/prometheus
```

Check if the service is available:

```sh
export NAMESPACE=<YOUR_NAMESPACE>

export POD_NAME=$(kubectl get pods -n $NAMESPACE -l "app=prometheus,component=server" -o jsonpath="{.items[0].metadata.name}")

kubectl --namespace $NAMESPACE port-forward $POD_NAME 9090:9090
```

```sh
helm install prometheus-adapter --set prometheus.url=http://prometheus-server.default.svc --set prometheus.port=80  prometheus-community/prometheus-adapter
```

### Deploy Prometheus adapter

Deploy the new prometheus adapter configuration to scrape custom metrics [configmap](k8s/prometheus-adapter/prometheus-adapter-cm.yaml)

Create a new prometheus adapter configuration configmap to scrape the new `active_connections` metric from go-demo-3 application:

```yaml
rules:
    - seriesQuery: '{__name__= "active_connections"}'
      resources:
        overrides:
          kubernetes_namespace:
            resource: namespace
          kubernetes_pod_name:
            resource: pod
      name:
        matches: "active_connections"
        as: ""
      metricsQuery: <<.Series>>{<<.LabelMatchers>>,container_name!="POD"}
```


```sh
kubectl apply -f k8s/prometheus-adapter/prometheus-adapter-cm.yaml
```

### Deploy Grafana

```sh
helm install grafana grafana/grafana
```

Connect to grafana:

```sh
export NAMESPACE=<YOUR_NAMESPACE>
export POD_NAME=$(kubectl get pods -n $NAMESPACE -l "app.kubernetes.io/name=grafana" -o jsonpath="{.items[0].metadata.name}")

kubectl --namespace $NAMESPACE port-forward $POD_NAME 3000:3000

```

Get admin password:

```sh
export NAMESPACE=<YOUR_NAMESPACE>

kubectl get secret -n $NAMESPACE grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
```

Open Grafana in your browser:

```sh
open http://localhost:3000
```

Your browser with open automatically Grafana.

## Annotations for scraping application metrics

To automatically scrape metrics from pods, add to the application with the following annotations:

```yaml
 annotations:
        prometheus.io/path: /metrics
        prometheus.io/port: "8080"
        prometheus.io/scrape: "true"
```

## Kubernetes HPA metrics

Check which custom metrics are avaible:

```sh
kubectl get --raw /apis/custom.metrics.k8s.io/v1beta1  | jq .
```

Get active connections:

```sh
kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/namespaces/default/pods/*/active_connections"  | jq
```