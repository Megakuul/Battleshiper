#!/bin/bash

set -e

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
echo "All required software is installed."


echo "Finding CloudFormation stacks with prefix 'battleshiper-'..."
stacks=$(aws cloudformation list-stacks --query "StackSummaries[?starts_with(StackName, 'battleshiper-') && StackStatus != 'DELETE_COMPLETE'].StackName" --output text)

if [[ -z "$stacks" ]]; then
  echo "No stacks found with prefix 'battleshiper-'."
else
  echo "The following stacks will be deleted:"
  echo "$stacks"
  confirm_action "Do you want to delete these stacks?"
  
  for stack in $stacks; Do
    set +e
    aws cloudformation delete-stack --stack-name $stack
    if [[ $? -eq 0 ]]; then
      echo "Stack deletion for $stack successfully initiated."
    else
      echo "Failed to delete stack $stack. Please check the AWS CloudFormation console for more information."
      exit 1
    fi
    echo "Waiting for stack $stack to be deleted..."

    aws cloudformation wait stack-delete-complete --stack-name $stack
    if [[ $? -eq 0 ]]; then
      echo "Stack $stack deleted successfully."
    else
      echo "Failed to delete stack $stack. Please check the AWS CloudFormation console for more information."
      exit 1
    fi
    set -e
  done
  echo "All stack deletions initiated."
fi

confirm_action "Do you want to delete the internal Battleshiper system?"
echo "Deleting the internal Battleshiper system..."
sam delete --stack-name battleshiper

echo "Battleshiper system deletion process complete."