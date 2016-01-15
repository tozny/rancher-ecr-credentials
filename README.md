# Rancher ECR Credentials Updater

This is Docker container that when executed will update the Docker registry credentials in Rancher for an Amazon Elastic Container Registry.

## Why is this needed?

Because access to ECR is controlled with AWS IAM.
An IAM user must request a temporary credential to the registry using the AWS API.
This temporary credential is then valid for 12 hours.

Rancher only supports registries that authenticate with a username and password.

## How to use

Run this container with the following environment variables:
* `AWS_REGION` - the AWS region of the ECR registry
* `AWS_ACCESS_KEY_ID`
* `AWS_SECRET_ACCESS_KEY`
* `RANCHER_URL` - the url of the Rancher server to update
* `RANCHER_ACCESS_KEY`
* `RANCHER_SECRET_KEY`

The AWS credentials must correspond to an IAM user that has permissions to call the ECR `GetToken` API.
The application then parses the resulting response to retrieve the ECR registry URL, username, and password.
The returned registry URL, is used to discover the corresponding registry in Rancher.

Rancher stores registries by environment.
If multiple environments exists, one instance of this container must be run per environment.
Rancher credentials are tied to an environment, so specifying them will indicate which environment to update in Rancher.

__NOTE__: This application runs on a 6 hour loop. It's possible there could be a slight gap where the credentials expire before this program updates them.
