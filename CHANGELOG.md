# ChangeLog

## v1.2.0 (Unreleased)

* Add HTTP healthcheck for container (default: :8080/ping, configurable with `LISTEN_PORT` envvar)
* Support multiple ECR registries with `AWS_ECR_REGISTRY_IDS` envvar
* Support auto creating ECR registry in Rancher using the `AUTO_CREATE` envvar (false by default)

## v1.1.0 (2016/07/21)

* Support IAM Instance Profiles for AWS API credentials

## v1.0.1 (2016/03/22)

* [bug] - don't exit process when error connecting to Rancher API

## v1.0.0 (2016/03/12)

* Initial Release
