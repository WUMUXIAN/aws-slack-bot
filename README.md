[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/WUMUXIAN/aws-slack-bot/blob/master/LICENSE)

### aws-slack-bot

This bot is implemented in golang and is meant to send AWS Usage report to an integrated slack channel. The following services are watched and reported:

1. EC2 Usage.
2. S3 Usage.
3. CloudFront Usage.
4. RDS Usage.
5. Elasticache Usage.
6. Estimated Billing.

![](https://github.com/WUMUXIAN/aws-slack-bot/blob/master/screenshots/part1.jpg)
![](https://github.com/WUMUXIAN/aws-slack-bot/blob/master/screenshots/part2.jpg)

You can specify any region(s) you want to watch and the frequency of reporting using `cron definition`

### Usage

The application is release via `Docker`, which can be found here: https://hub.docker.com/r/wumuxian/aws-slack-bot/
Choose a version to use, currently the latest version is `wumuxian/aws-slack-bot:v0.0.1`

You need to set the following parameters to make it work.

| Environment Variables |                               Decription                               |
|:---------------------:|:----------------------------------------------------------------------:|
|   AWS_ACCESS_KEY_ID   |                  You access key ID to the AWS account                  |
| AWS_SECRET_ACCESS_KEY |                  You access secret to the AWS account                  |
|     AWS_ACCOUNT_ID    |                  The AWS account ID you want to watch                  |
|   SLACK_WEBHOOK_URL   |                       The slack channel web-hook                       |
|    CRON_DEFINITION    | The cron job definition that sets the frequency of the report          |
|        REGIONS        | Comma separated regions you want to watch, e.g. us-east-1,eu-central-1 |

> Notes: Please note that your IAM must be granted relevant read access to the services.
> If not then you won't receive report for that service.

If you don't specify the `CRON_DEFINITION`, the default will be `0 0 1 * * MON-FRI`, which means "Every 9am in the morning, from Monday to Friday, SGT Timezone"

If you don't specify the `REGIONS`, the default will be `us-east-1`, which is `US East (N. Virginia)`

The `SLACK_WEBHOOK_URL` is a required field, if not specified, the program won't run.

The `CRON_DEFINITION` must have correct cron syntax, otherwise the program won't run.

Example execution script using docker:

```bash
docker run -d \
  -e AWS_ACCESS_KEY_ID="" \
  -e AWS_SECRET_ACCESS_KEY="" \
  -e AWS_ACCOUNT_ID = "" \
  -e SLACK_WEBHOOK_URL="" \
  -e CRON_DEFINITION="0/5 * * * * ?" \
  -e REGIONS="us-east-1,ap-southeast-1" \
  --name aws-slack-bot wumuxian/aws-slack-bot:v0.0.1
```

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
go run *.go
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
