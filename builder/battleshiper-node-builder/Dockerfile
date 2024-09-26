FROM amd64/amazonlinux:2023

# Install required build tools
RUN yum update -y && yum install -y aws-cli git

# Install node runtime
RUN yum install -y nodejs