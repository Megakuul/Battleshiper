# DELETE

This document describes how to remove the battleshiper system.

Alternatively, you can use the [Delete Script](/scripts/delete.sh) to guide you through the process.

The instructions below require the following software packages to be installed on your system:
- `aws cli`
- `aws sam cli`


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