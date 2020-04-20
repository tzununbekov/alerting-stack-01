# alerting-stack-01

Simple [triggerflow](triggerflow.yaml) that creates Knative components to retrieve Prometheus data, 
process it and send Slack message if some numbers are fits the 
[logic](main.go#L62)
