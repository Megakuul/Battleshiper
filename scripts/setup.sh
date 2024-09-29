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


request_certificate() {
    local domain=$1
    echo "Requesting ACM certificate for $domain..."
    cert_arn=$(aws acm request-certificate --region us-east-1 --domain-name "$domain" --validation-method DNS --query 'CertificateArn' --output text)
    echo "ACM certificate ARN: $cert_arn"
    echo "Waiting for certificate..."
    sleep 3
    aws acm describe-certificate --region us-east-1 --certificate-arn "$cert_arn" --query 'Certificate.DomainValidationOptions[0].ResourceRecord'
    echo "Add the above DNS record to your domain."
}

request_wild_certificate() {
    local domain=$1
    echo "Requesting ACM certificate for $domain..."
    cert_wild_arn=$(aws acm request-certificate --region us-east-1 --domain-name "$domain" --validation-method DNS --query 'CertificateArn' --output text)
    echo "ACM certificate ARN: $cert_wild_arn"
    echo "Waiting for certificate..."
    sleep 3
    aws acm describe-certificate --region us-east-1 --certificate-arn "$cert_wild_arn" --query 'Certificate.DomainValidationOptions[0].ResourceRecord'
    echo "Add the above DNS record to your domain."
}

# Function to create GitHub credentials in AWS Secrets Manager
create_github_secrets() {
    secret_name="battleshiper-github-credentials"

    echo "Checking if the secret already exists..."

    set +e
    github_cred_arn=$(aws secretsmanager describe-secret --secret-id $secret_name --query 'ARN' --output text)
    set -e

    if [ -n "$github_cred_arn" ]; then
        read -p "Skip secret update (y/N): " choice
        case "$choice" in 
            y|Y ) echo "Skipping..."; return 0;;
        esac
    fi

    read -p "Enter GitHub Client ID: " client_id
    read -p "Enter GitHub Client Secret: " client_secret
    read -p "Enter GitHub App ID: " app_id
    read -p "Enter GitHub App Secret (private key): " app_secret
    read -p "Enter GitHub Webhook Secret: " webhook_secret

    if [ -z "$github_cred_arn" ]; then
        echo "Creating GitHub credentials in AWS Secrets Manager..."
        aws secretsmanager create-secret \
            --name $secret_name \
            --secret-string "{\"client_id\":\"$client_id\",\"client_secret\":\"$client_secret\",\"app_id\":\"$app_id\",\"app_secret\":\"$app_secret\",\"webhook_secret\":\"$webhook_secret\"}" \
            --query 'ARN' --output text
        echo "GitHub Credentials ARN: $github_cred_arn"
    else
        echo "Secret already exists. Updating the secret..."
        aws secretsmanager update-secret \
            --secret-id $secret_name \
            --secret-string "{\"client_id\":\"$client_id\",\"client_secret\":\"$client_secret\",\"app_id\":\"$app_id\",\"app_secret\":\"$app_secret\",\"webhook_secret\":\"$webhook_secret\"}" \
            --query 'ARN' --output text
        echo "GitHub Credentials updated. ARN: $github_cred_arn"
    fi
}

# Step 1: Request ACM Certificates
read -p "Enter your Battleshiper domain (e.g., battleshiper.dev): " domain
request_certificate "$domain"
confirm_action "Have you added the DNS record for the base domain?"

# Request wildcard certificate
wild_domain="*.$domain"
request_wild_certificate "$wild_domain"
confirm_action "Have you added the DNS record for the wildcard domain?"

# Step 2: Set up GitHub Application and Credentials
echo "Set up the GitHub application following the GitHub documentation:"
echo " - Set Callback URL to https://$domain/api/auth/callback"
echo " - Set Webhook URL to https://$domain/api/pipeline/event"
confirm_action "Have you created the GitHub application and extracted credentials?"

echo "Generating GitHub application secret..."
create_github_secrets


read -p "Enter the GitHub username that will be selected as admin: " username

# Step 3: Build and Deploy the Battleshiper System
echo "Building the Battleshiper system with AWS SAM..."
sam build

echo "Deploying the Battleshiper system to AWS..."
sam deploy --parameter-overrides ApplicationDomain="$domain" ApplicationDomainCertificateArn="$cert_arn" ApplicationDomainWildcardCertificateArn="$cert_wild_arn" GithubOAuthClientCredentialArn="$github_cred_arn" GithubAdministratorUsername="$username"

cdn_host=$(aws cloudformation describe-stacks --stack-name battleshiper --query "Stacks[0].Outputs[?OutputKey=='BattleshiperCDNHost'].OutputValue" --output text)
cdn_project_host=$(aws cloudformation describe-stacks --stack-name battleshiper --query "Stacks[0].Outputs[?OutputKey=='BattleshiperProjectCDNHost'].OutputValue" --output text)


# Step 4: Final DNS Setup
echo "Add the following DNS records to your provider to finalize deployment:"
echo "1. CNAME $domain $cdn_host"
echo "2. CNAME *.$domain $cdn_project_host"

echo "Battleshiper system setup complete."