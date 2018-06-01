#!/bin/bash

currentVersion=1
if [ -f "currentVersion" ];then
	currentVersion=`cat currentVersion`
	((currentVersion=currentVersion+1))
fi
echo $currentVersion > currentVersion

echo "Deploying Version: v$currentVersion"
docker tag wumuxian/aws-slack-bot wumuxian/aws-slack-bot:v$currentVersion
docker push wumuxian/aws-slack-bot:v$currentVersion
