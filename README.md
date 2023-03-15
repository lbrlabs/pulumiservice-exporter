# pulumiservice-exporter

> :warning: **This is a work in progress**: Any expectations of production readiness fall entirely on the user

A Prometheus exporter for the Pulumi Service

## Quick Start 

You can run the exporter a Docker image or as a standalone binary.

Simply specify the Pulumi org and an access token

``
pulumiserver-export --org=demo --access-token=pul-xxxxxx
```

```
docker run -e PULUMI_ORG=${PULUMI_ORG} -e PULUMI_ACCESS_TOKEN=${PULUMI_ACCESS_TOKEN} ghcr.io/lbrlabs/pulumiservice-exporter
```
