# UPDATE

This document describes how to update the battleshiper system.

Alternatively, you can use the [Update Script](/scripts/update.sh) to guide you through the process.

The instructions below require the following software packages to be installed on your system:
- `aws cli`
- `aws sam cli`
- `bun`
- `go`


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