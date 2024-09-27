#!/bin/bash

set -e

cd "$(dirname "$0")/.."

check_command() {
  if ! command -v $1 &> /dev/null; then
    echo "$1 is required but not installed. Please install it before proceeding."
    exit 1
  fi
}

confirm_action() {
    read -p "$1 (y/N): " choice
    case "$choice" in 
        y|Y ) echo "Proceeding...";;
        * ) echo "Aborting."; exit 0;;
    esac
}

echo "Checking required software..."
check_command "aws"
check_command "sam"
check_command "bun"
check_command "go"
echo "All required software is installed."

echo "Building the SAM stack..."
sam build

confirm_action "Do you want to proceed with deploying the updated SAM stack?"
echo "Deploying the updated SAM stack..."
sam deploy

web_bucket=$(aws cloudformation describe-stacks --stack-name battleshiper --query "Stacks[0].Outputs[?OutputKey=='BattleshiperWebBucket'].OutputValue" --output text)

echo "Removing previous static assets from the web bucket..."
aws s3 rm s3://"$web_bucket"/ --recursive

echo "Uploading updated static assets for the Battleshiper dashboard..."
aws s3 cp --recursive .aws-sam/build/BattleshiperApiWebFunc/prerendered/ s3://"$web_bucket"/
aws s3 cp --recursive .aws-sam/build/BattleshiperApiWebFunc/client/ s3://"$web_bucket"/

echo "Update process complete."