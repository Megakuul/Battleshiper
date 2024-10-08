# SETUP

This document describes how to setup and install the battleshiper system.

Alternatively, you can use the [Setup Script](/scripts/setup.sh) to guide you through the process.

The instructions below require the following software packages to be installed on your system:
- `aws cli`
- `aws sam cli`
- `bun`
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
- Enable "User-to-server token expiration" feature.
- Set permission of "Repository->Contents" to "read-only".
- Subscribe to "Push" and "Repository" events.
- Enable Webhook and set the URL to https://$DOMAIN/api/pipeline/event.
- Create a strong Webhook secret and remember it for the next step.

Finally, create and extract the following credentials:
- `client_id`
- `client_secret`
- `app_id`
- `app_secret` (labelled as "private key")
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

Notice that the `app_secret` must be in the base64 pem format, **OMITTING** the "-----BEGIN RSA PRIVATE KEY-----" and "-----END RSA PRIVATE KEY-----" specifiers, as AWS Secrets Manager cannot handle newline characters (\n).




Finally extract the ARN, which will be used later for deployment.



## Deployment
---
Battleshiper can be deployed with the `aws sam cli`.


### Build

First compile the lambda and web functions:
```bash
sam build
```

(data displayed on the frontend (e.g. link to the github app) can be customized in the `web/.env` file)


### Deploy

Then deploy them to aws with deploy:
```bash
sam deploy --parameter-overrides ApplicationDomain=$DOMAIN ApplicationDomainCertificateArn=$CERT_ARN ApplicationDomainWildcardCertificateArn=$WILD_CERT_ARN GithubOAuthClientCredentialArn=$GITHUB_CRED_ARN GithubAdministratorUsername=Megakuul
```

Specify your Github username as `GithubAdministratorUsername`. Doing so will grant your account the `ROLE_MANAGER` role during registration.
This is the highest privilege role, allowing you to assign all other roles to your account.


Uploading the Sveltekit assets of the webdashboard is integrated into the CodeDeploy process. Initially, the assets are not uploaded because Cloudformation skips the CodeDeploy pipeline. For this reason, you must once redeploy the application to correctly build and upload the full web dashboard (note that this step only updates the web components and does not redeploy the entire application):
```bash
sam build && sam deploy
``` 

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

Don't worry if Battleshiper doesn't work as expected right away. As explained in [this](https://stackoverflow.com/questions/38735306/aws-cloudfront-redirecting-to-s3-bucket) article, it can take some time for S3 DNS settings to fully propagate. During this time, Cloudfront might incorrectly redirect requests to the S3 bucket instead of serving the content directly.