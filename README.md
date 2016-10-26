### aws-slack-bot

This bot is implemented in GO and is meant to send AWS Usage report to an integrated slack channel every day at 9:00am (Singapore time) from Monday to Friday. For now the following information is reported:

1. Number of running instances. (us-east-1)
2. Estimated cost for current month.

#### Dependencies
* [Go](https://golang.org/doc/install) 
* [Glide](https://github.com/Masterminds/glide)

#### Setup
```bash
glide install 
glide up
```

#### Run locally (assume you're using Mac)
```bash
./build_and_run_local
```

#### Build a linux runnable
```bash
./build_linux
```