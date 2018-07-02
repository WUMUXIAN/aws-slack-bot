#!/bin/bash

####################################################################
# This script automatically bump the tagged version and release it.
# The version is tagged in git with format releases/v[x].[y].[z]
# x, y, z are single digit numbers from 0-9.
# Please only call it when you have a stable release.
####################################################################

# get the current version.
current_version=$(git tag -l "releases/*" --sort=-v:refname | head -n 1)
if [ "$current_version" = "" ];then
	current_version="0.0.0"
else
	current_version=${current_version#"releases/v"}
fi

# bump the version number
echo "current version is ${current_version}"
versions=(${current_version//./ })
x=${versions[0]}
y=${versions[1]}
z=${versions[2]}

z=$((z + 1))
if [ $z -eq 10 ];then
    y=$((y + 1))
    z=0
fi

if [ $y -eq 10 ];then
    x=$((x + 1))
    y=0
fi

new_version="${x}.${y}.${z}"
echo "new version is ${new_version}"

echo "tagging git with new version and push"
# tag the commit with the new tag and release the docker image.
git tag -a releases/v${new_version} -m "releases/v${new_version}"
# push the tag
git push origin releases/v${new_version}

hash=$(git rev-parse --verify --short HEAD)

# Tag the docker image
docker tag wumuxian/aws-slack-bot wumuxian/aws-slack-bot:v${new_version}

echo "pushing images"
docker push wumuxian/aws-slack-bot:latest
docker push wumuxian/aws-slack-bot:v${new_version}

# Cleanup
images=$(docker images -q --filter "dangling=true")
echo $images
if [ "$images" != "" ]; then
    docker rmi $images
fi
