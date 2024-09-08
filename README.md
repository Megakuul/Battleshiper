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
To activate the certificate, you must add the specified DNS records to your domain provider.



### GitHub Application
Create the application via the GitHub interface. If you need guidance, refer to their [documentation](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app).

While creating the app, it's important to configure the following parameters:
- 
- 
- 

Finally, create and extract the following credentials:
- `app_id`
- `app_secret`
- `client_id`
- `client_secret`



### GitHub Application Credentials
To make the credentials accessible to your system, create a secret in AWS Secrets Manager containing the extracted credentials.

You can use the `aws cli` to do this:
```bash
aws secretsmanager create-secret \
    --name battleshiper-github-credentials \
    --secret-string '{"app_id":"1234","app_secret":"1234","client_id":"1234","client_secret":"1234"}' \
    --query 'ARN' --output text
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
sam deploy --parameter-overrides ApplicationDomain=$DOMAIN ApplicationDomainCertificateArn=$CERT_ARN GithubOAuthClientCredentialArn=$GITHUB_CRED_ARN
```

## Finalize
---


## Update
---
If you want to update the Battleshiper system, you can simply update the sam stack and then redeploy it:
```bash
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