# UserClouds Headless Container

The UserClouds Headless Container is a Docker container that can be used to test the UserClouds platform.
It runs 3 services (plex, authz and userstore) and uses a postgres DB to store data.
We use a [container runner tool](../../cmd/containerrunner/README.md) inside the container to run services, route traffic to them and apply a config on startup.

You can use the [helper script](../../tools/packaging/build-and-run-userclouds-headless.sh) to build and run the container using the [docker compose file](docker-compose.yml).
Once the container is up and running it it can be accessed at the tenant URL: <http://container-dev.tenant.test.userclouds.tools:3040> note that there is no web UI (console) so the container is only accessible via APIs.
connection details:

- Tenant URL: <http://container-dev.tenant.test.userclouds.tools:3040>
- client ID: `container_dev_client_id`
- client Secret: `container_dev_client_secret_fake_dont_use_outside_of_dev`
- Tenant ID: `11111111-dbe0-4f21-b3c9-9bc29f0bae03`

All of these details are coming from [the provisioning file](../../config/provisioning/samples/company_uc_container_dev.json) we use to provision the tenant in the container.
