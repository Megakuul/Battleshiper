FROM amd64/amazonlinux:2023

# Install required build tools
RUN yum update -y && yum install -y aws-cli git unzip

# Install bun runtime
RUN export BUN_INSTALL="/usr/local" && curl -fsSL https://bun.sh/install | bash