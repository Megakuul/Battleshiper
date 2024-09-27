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


Finally some static assets and prerendered pages of the internal battleshiper dashboard must be uploaded.

Unfortunately, there is no really clean way to handle this situation with aws sam, for that reason, the assets are written to the sam output directory:
```bash
# Upload static assets for the battleshiper dashboard
aws s3 cp --recursive .aws-sam/build/BattleshiperApiWebFunc/prerendered/ s3://"$web_bucket"/
aws s3 cp --recursive .aws-sam/build/BattleshiperApiWebFunc/client/ s3://"$web_bucket"/
```
(The bucketname can be acquired from the sam deploy output).

Note that when you deploy the app, the assets must be updated at the same time. Svelte adds hashes to the js chunks, so the server version must match the assets!