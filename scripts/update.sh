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

echo "Update process complete."