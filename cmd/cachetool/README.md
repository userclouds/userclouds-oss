# Cache management tool

This tool can be used to manage various caches, including clearing and logging key values.
Currently, it supports the following caches:

* AuthZ - can be cleared by tenant or for all tenants
* UserStore - can be cleared by tenant or for all tenants
* Plex Map Cache
* Company Config Cache

## Implementation

The tool sends messages to the worker, which in turn runs the [logic](../../worker/internal/cachetool/) to execute the supported command.

## Usage

Build the tool: `make bin/cachetool`

Run the tool:

```shell
UC_UNIVERSE=<debug, staging or prod> UC_REGION="aws-us-east-1" bin/cachetool <command> <cache-name> [tenant-id]

To run the command for all tenants (applicable for caches like AuthZ and UserStore), use `all` as the tenant ID. This triggers the tool to enumerate all active tenants and send a cache clearing command to each.
In this scenario, the tool will paginate through all tenants and send a clear cache message for each tenant.
