### aws-slack-bot

This bot is implemented in GO and is meant to send AWS Usage report to an integrated slack channel every day at 9:00am (Singapore time) from Monday to Friday. For now the following information is reported:

1. Number of running instances. (us-east-1)
2. Estimated cost for current month.

#### Dependencies
* [Go](https://golang.org/doc/install) 
* [Glide](https://github.com/Masterminds/glide)

#### Setup
```bash
go get -u github.com/aws/aws-sdk-go
glide install 
glide up
```

#### Slack Integration
1. Set up your slack webhook and get the URL.
2. Create a file namely *slack_webhook_url* (case sensitive) and put it at the same path as the runnable.

#### Run locally (assume you're using Mac)
```bash
./build_and_run_local
```

#### Build a linux runnable
```bash
./build_linux
```

#### Deploy
It takes a single runnable file and a slack_webhook_url file to deploy. Make sure that you put the slack_webhook_url at your system's $HOME directory in order to let the application read it correctly.