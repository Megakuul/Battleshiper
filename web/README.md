# web

the web directory holds the battleshiper ui, it is deployed as the first project on the battleshiper system itself.
there are no special permissions etc., the app just uses the public battleshiper api.


variables to customize the webapp are defined in the `.env` file, which is evaluated at build time and statically injected into the project.
before building, copy the `.env.example` file and adjust your website configuration.