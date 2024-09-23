# SETUP

This document describes how to setup and install the battleshiper system.

Alternatively, you can use the [Setup Script](/scripts/setup.sh) to guide you through the process.

The instructions below require the following software packages to be installed on your system:
- `aws cli`
- `aws sam cli`
- `nodejs`
- `go`


## Preparation
---
To set up Battleshiper, the following prerequisites are required:
- `active aws acm certificate`
- `github application`
- `github application credentials`


### ACM Certificate
You can use the `aws cli` to generate an ACM certificate for the desired Battleshiper domain:
```bash
export DOMAIN="battleshiper.dev"
export CERT_ARN=$(aws acm request-certificate --region us-east-1 --domain-name $DOMAIN --validation-method DNS --query 'CertificateArn' --output text)
aws acm describe-certificate --region us-east-1 --certificate-arn $CERT_ARN --query 'Certificate.DomainValidationOptions[0].ResourceRecord'
```

Next create a wildcard certificate on your domain, this certificate will be used for the projects hosted on battleshiper:
```bash
export WILD_CERT_ARN=$(aws acm request-certificate --region us-east-1 --domain-name "*.$DOMAIN" --validation-method DNS --query 'CertificateArn' --output text)
aws acm describe-certificate --region us-east-1 --certificate-arn $WILD_CERT_ARN --query 'Certificate.DomainValidationOptions[0].ResourceRecord'
```

To activate the certificates, you must add the specified DNS records to your domain.


### GitHub Application
Create the application via the GitHub interface. If you need guidance, refer to their [documentation](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app).

While creating the app, it's important to configure the following parameters:
- Set Callback URL to https://$DOMAIN/api/auth/callback.
- Enable Webhook and set the URL to https://$DOMAIN/api/pipeline/event.
- Create a strong Webhook secret and remember it for the next step.
- 

Finally, create and extract the following credentials:
- `client_id`
- `client_secret`
- `app_id`
- `app_secret`
- `webhook_secret`



### GitHub Application Credentials
To make the credentials accessible to your system, create a secret in AWS Secrets Manager containing the extracted credentials.

You can use the `aws cli` to do this:
```bash
export GITHUB_CRED_ARN=$(aws secretsmanager create-secret \
    --name battleshiper-github-credentials \
    --secret-string '{"client_id":"1234","client_secret":"1234","app_id":"12345","app_secret":"12345","webhook_secret":"1234"}' \
    --query 'ARN' --output text)
```

Finally extract the ARN, which will be used later for deployment.



## Deployment
---
Battleshiper can be deployed with the `aws sam cli`.


### Build

First compile the lambda functions:
```bash
sam build
```


### Deploy

Then deploy them to aws with deploy:
```bash
sam deploy --parameter-overrides ApplicationDomain=$DOMAIN ApplicationDomainCertificateArn=$CERT_ARN ApplicationDomainWildcardCertificateArn=$WILD_CERT_ARN GithubOAuthClientCredentialArn=$GITHUB_CRED_ARN GithubAdministratorUsername=Megakuul
```

Specify your Github username as `GithubAdministratorUsername`. Doing so will grant your account the `ROLE_MANAGER` role during registration.
This is the highest privilege role, allowing you to assign all other roles to your account.


### Upload Assets

Finally some static assets and prerendered pages of the internal battleshiper dashboard must be uploaded.
The following assets must be placed in the specified bucket location:
- `web/build/prerendered/*` -> `$BattleshiperWebBucket/`
- `web/build/client/*` -> `$BattleshiperWebBucket/`
- `web/404.html` -> `$BattleshiperProjectWebBucket/404.html`

This can be done by first generating the build files (if not existent) and then sending the files via s3:
```bash
cd web
npm ci && npm run build

# Upload 404 page to project bucket
s3 cp 404.html s3://$BattleshiperProjectWebBucket/404.html
# Upload static assets for the battleshiper dashboard
s3 cp --recursive build/prerendered/ s3://$BattleshiperWebBucket/
s3 cp --recursive build/client/ s3://$BattleshiperWebBucket/
```

(The bucketnames can be acquired from the sam deploy output).


## Finalize
---

To finalize the deployment, first you need to add two dns records to your provider:

1. For the base domain, set the CNAME to the battleshiper cdn hostname found in the sam output:
```bash
CNAME $DOMAIN <BATTLESHIPER-CDN-HOST>
```

2. For the wildcard domain, set the CNAME to the battleshiper project cdn hostname found in the sam output:
```bash
CNAME *.$DOMAIN <BATTLESHIPER-PROJECT-CDN-HOST>
```