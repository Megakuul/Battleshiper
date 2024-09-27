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
check_command "bun"
check_command "go"
echo "All required software is installed."


request_certificate() {
    local domain=$1
    echo "Requesting ACM certificate for $domain..."
    cert_arn=$(aws acm request-certificate --region us-east-1 --domain-name "$domain" --validation-method DNS --query 'CertificateArn' --output text)
    echo "ACM certificate ARN: $cert_arn"
    aws acm describe-certificate --region us-east-1 --certificate-arn "$cert_arn" --query 'Certificate.DomainValidationOptions[0].ResourceRecord'
    echo "Add the above DNS record to your domain."
}

# Function to create GitHub credentials in AWS Secrets Manager
create_github_secrets() {
    read -p "Enter GitHub Client ID: " client_id
    read -p "Enter GitHub Client Secret: " client_secret
    read -p "Enter GitHub Webhook Secret: " webhook_secret

    echo "Creating GitHub credentials in AWS Secrets Manager..."
    github_cred_arn=$(aws secretsmanager create-secret \
        --name battleshiper-github-credentials \
        --secret-string "{\"client_id\":\"$client_id\",\"client_secret\":\"$client_secret\",\"webhook_secret\":\"$webhook_secret\"}" \
        --query 'ARN' --output text)
    echo "GitHub Credentials ARN: $github_cred_arn"
}

# Step 1: Request ACM Certificates
read -p "Enter your Battleshiper domain (e.g., battleshiper.dev): " domain
request_certificate "$domain"
confirm_action "Have you added the DNS record for the base domain?"

# Request wildcard certificate
wild_domain="*.$domain"
request_certificate "$wild_domain"
confirm_action "Have you added the DNS record for the wildcard domain?"

# Step 2: Set up GitHub Application and Credentials
echo "Set up the GitHub application following the GitHub documentation:"
echo " - Set Callback URL to https://$domain/api/auth/callback"
echo " - Set Webhook URL to https://$domain/api/pipeline/event"
confirm_action "Have you created the GitHub application and extracted credentials?"

create_github_secrets

# Step 3: Build and Deploy the Battleshiper System
echo "Building the Battleshiper system with AWS SAM..."
sam build

echo "Deploying the Battleshiper system to AWS..."
sam deploy --parameter-overrides \
    ApplicationDomain="$domain" \
    ApplicationDomainCertificateArn="$cert_arn" \
    ApplicationDomainWildcardCertificateArn="$wild_cert_arn" \
    GithubOAuthClientCredentialArn="$github_cred_arn" \
    GithubAdministratorUsername="YourGitHubUsername"

# Step 4: Upload Static Assets
echo "Uploading static assets..."
cd web
bun install && bun run build

read -p "Enter BattleshiperWebBucket name from SAM deploy output: " web_bucket
read -p "Enter BattleshiperProjectWebBucket name from SAM deploy output: " project_web_bucket

aws s3 cp 404.html s3://"$project_web_bucket"/404.html
aws s3 cp --recursive build/prerendered/ s3://"$web_bucket"/
aws s3 cp --recursive build/client/ s3://"$web_bucket"/

# Step 5: Final DNS Setup
echo "Add the following DNS records to your provider to finalize deployment:"
echo "1. CNAME $domain <BATTLESHIPER-CDN-HOST>"
echo "2. CNAME *.$domain <BATTLESHIPER-PROJECT-CDN-HOST>"

echo "Battleshiper system setup complete."