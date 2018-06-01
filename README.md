### aws-slack-bot

This bot is implemented in golang and is meant to send AWS Usage report to an integrated slack channel every day at 9:00am (UTC+8 TZ) from Monday to Friday. For now the following information is reported:

1. EC2 Usage.
2. S3 Usage.
3. CloudFront Usage.
4. RDS Usage.
5. Elasticache Usage.
6. Estimated Cost.

At the moment, it supports us-east-1 only, it will be changed to adopt any region you specify in the future.

### Usage

The docker image is readily for use at `wumuxian/aws-slack-bot:v2`.
Specify your AWS credentials, region and your slack integration channel to make it work.

```bash
docker run -d -e AWS_ACCESS_KEY_ID="" -e AWS_SECRET_ACCESS_KEY="" -e AWS_ACCOUNT_ID = "" -e SLACK_WEBHOOK_URL="" --name aws-slack-bot wumuxian/aws-slack-bot:v2
```

>Notes: You will have to configure the permissions of your account to have read access to
  - EC2
  - Elasticache
  - S3
  - CloudFront
  - RDS
  - Cloundwatch

### Development

If you want to contribute to the repo, please continue to read.

#### Dependencies
* [Go](https://golang.org/doc/install)
* [Dep](https://github.com/golang/dep)

#### Setup
```bash
dep init
dep ensure
```

#### Run locally
```bash
./run.sh
```

#### Build docker image
```bash
./build.sh
```
> Feel free to build to your own images, just modify the build.sh script.

#### Release
```bash
./release.sh
```
> Fee free to release to your own repo, just modify the release.sh script
