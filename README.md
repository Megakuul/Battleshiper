# Battleshiper

Battleshiper - A Serverless Sveltekit Deployment Platform. 

![battleshiper favicon](/web/static/favicon.png "battleshiper favicon")


[How to setup Battleshiper?](/docs/SETUP.md)

[How to update Battleshiper?](/docs/UPDATE.md)

[How to delete Battleshiper?](/docs/DELETE.md)


## Architecture
---

Battleshiper is composed of three core systems.
- Internal
- Project
- Pipeline

All of those systems serve specific use cases that are explained below.


### Internal System

![internal system architecture](/docs/assets/battleshiper_internal.png)
(example is illustrative and not fully comprehensive)



The internal system primarily consists of the Battleshiper API, which serves as the interface for the entire application.

In addition to the API, the system includes a DynamoDB database (painfully migrated from DocumentDB), which stores all user, subscription, and project data. To provide a dashboard, the internal system uses a custom CloudFront instance in combination with an S3 bucket to host the web dashboard (which is controlled by a catch-all API SvelteKit server function).


### Project System

![project system architecture](/docs/assets/battleshiper_project.png)
(example is illustrative and not fully comprehensive)



The project system is the core product of Battleshiper, providing the infrastructure that powers customer projects.

This system consists of a core CloudFront instance used for all customer projects. The structure of SvelteKit applications is leveraged to create a highly efficient system: All static assets (`/_app/*`) and prerendered pages are stored in an S3 bucket. A CloudFront Function (CacheRouteFunc) routes requests to the corresponding bucket path based on the requested hostname.

Requests for static assets are cached after being fetched once, utilizing CloudFronts native caching mechanisms.

Traffic for non-static content is sent to a custom router Lambda function, which directly invokes the corresponding project function. CloudFront is connected to the Lambda via an API Gateway that redirects all traffic to the router. Initially, the API Gateway was responsible for routing, but due to its limitations, a custom router function was added.

To inform the router about which project it must route to, another CloudFront Function (ServerRouteFunc) adds the requested project as a custom header to the request.

Finally, there are specific considerations for prerendered pages. If a user requests a prerendered page without specifying the `.html` extension, CloudFront cannot identify it as prerendered. To resolve this, entries for all prerendered pages are stored in a CloudFront edge cache (Route Cache). The function checks these entries and, on a match, manually adds the .html extension.


### Pipeline System

![pipeline system architecture](/docs/assets/battleshiper_pipeline.png)
(example is illustrative and not fully comprehensive)


The pipeline system is the backbone of Battleshiper, used to build, deploy, and control projects.

The core product of this system is an API controlled by a central EventBridge bus. The API functions are used to initialize, deploy, and delete projects. Additionally, a Batch Job Queue is employed to build the projects.

To ensure the user-defined build process is fully isolated, a custom VPC is dedicated to the build Batch Jobs. During the build process, each project is granted permission to write data to a specific prefix in the build asset S3 bucket, where data is automatically cleaned up after a few days. The build assets are later validated and transferred from this bucket into the project system.

For added security, the execution of API functions requires a ticket. This ticket contains details about the source, destination, project, and user involved, and is signed with a key stored in SecretsManager. This mechanism ensures that the execution of pipeline functions is not solely restricted by IAM permissions to the event bus.