# Battleshiper


## Preparation
---
To set up Battleshiper, the following prerequisites are required:
- `aws cli`
- `aws sam cli`
- `active aws acm certificate`
- `github application`
- `github application credentials`

### CLI Tools 
You can install the cli tools with your local package manager:
```bash
yay -S aws-cli-v2 aws-sam-cli
```



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
- `webhook_secret`



### GitHub Application Credentials
To make the credentials accessible to your system, create a secret in AWS Secrets Manager containing the extracted credentials.

You can use the `aws cli` to do this:
```bash
export GITHUB_CRED_ARN=$(aws secretsmanager create-secret \
    --name battleshiper-github-credentials \
    --secret-string '{"client_id":"1234","client_secret":"1234","webhook_secret":"1234"}' \
    --query 'ARN' --output text)
```

Finally extract the ARN, which will be used later for deployment.



## Deployment
---
Battleshiper can be deployed with the `aws sam cli`.

First compile the lambda functions:
```bash
sam build
```

Then deploy them to aws with deploy:
```bash
sam deploy --parameter-overrides ApplicationDomain=$DOMAIN ApplicationDomainCertificateArn=$CERT_ARN ApplicationDomainWildcardCertificateArn=$WILD_CERT_ARN GithubOAuthClientCredentialArn=$GITHUB_CRED_ARN GithubAdministratorUsername=Megakuul
```

Specify your Github username as `GithubAdministratorUsername`. Doing so will grant your account the `ROLE_MANAGER` role during registration.
This is the highest privilege role, allowing you to assign all other roles to your account.


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


## Update
---
If you want to update the Battleshiper system, you can simply update the sam stack and then redeploy it:
```bash
sam build

sam deploy
```

**IMPORTANT**:
- You can only update the internal Battleshiper components, project stacks must be updated manually if necessary.
- If you update the system you must ensure that all updated properties can be "updated" by cloudformation.


## Delete
---
To fully remove the Battleshiper system you first need to delete all project stacks.
Those stacks can be found in the cloudformation console and can simply be deleted.

After all project stacks are cleaned up, you can delete the internal Battleshiper system with the sam cli:
```bash
sam delete --stack-name battleshiper
```

**IMPORTANT**:
If deletion fails due to dependencies on VPC components, make sure to delete all lambda network interfaces (ENIs). Since ENIs are managed by lambda, not cloudformation, there can be a slight delay in their removal, as noted [here](https://stackoverflow.com/questions/41299662/aws-lambda-created-eni-not-deleting-while-deletion-of-stack), which may cause issues.