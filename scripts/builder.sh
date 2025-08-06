#!/bin/bash
###
# Licensed Materials - Property of PEG TECH INC
#
# (C) Copyright PEG TECH INC. 2019 -2020All Rights Reserved
#
# Contributors:
#    bryan@raksmart.com - Initial implementation
#
#   Purpose:  set up the go env and build the binary
#
###

#set -e

GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
# 获取最新的 Git 标签
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null)
# 如果没有标签，默认从 v1.0.0 开始
if [ -z "$CURRENT_VERSION" ]; then
    if [ "${GIT_BRANCH}" == "master" ]; then
        CURRENT_VERSION="v1.0.0"
    else
        CURRENT_VERSION="staging-v1.0.0"
    fi
fi
VERSION_PREFIX="v"
TAG_PREFIX="v"
if [ "${GIT_BRANCH}" != "master" ]; then
    VERSION_PREFIX="${GIT_BRANCH}-v"
    TAG_PREFIX="staging-v"
fi
# 解析版本号
IFS='.' read -r MAJOR MINOR PATCH <<< "${CURRENT_VERSION/${TAG_PREFIX}/}"
# 增加修订号
PATCH=$((PATCH + 1))
# 生成下一个版本号
NEXT_VERSION="${VERSION_PREFIX}${MAJOR}.${MINOR}.${PATCH}"
# 输出下一个版本号
echo "Next version: $NEXT_VERSION"

make GIT_BRANCH=${GIT_BRANCH} VERSION=${NEXT_VERSION} build

image_name=$(cat image_name)
image_latest=$(cat image.latest)

# clean workdir
rm -rf ${WORKDIR}

# push image with tag
docker tag ${image_latest} reg-staging.raksmart.com:8443/petacloud/${image_latest} && \
    docker login -u petacloud -p Adc2tek! reg-staging.raksmart.com:8443 && \
    docker push reg-staging.raksmart.com:8443/petacloud/${image_latest} && \
    docker rmi reg-staging.raksmart.com:8443/petacloud/${image_latest}

# push image with tag latest
docker tag ${image_latest} reg-staging.raksmart.com:8443/petacloud/${image_name} && \
    docker login -u petacloud -p Adc2tek! reg-staging.raksmart.com:8443 && \
    docker push reg-staging.raksmart.com:8443/petacloud/${image_name} && \
    docker rmi reg-staging.raksmart.com:8443/petacloud/${image_name}

# clean up
docker rmi ${image_latest}

echo "Image ${image_name} pushed to reg-staging.raksmart.com:8443/petacloud/${image_name}"

