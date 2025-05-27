# UserClouds Command Line Tools Helpers

This package implements a set of helper functions intended to use on our various command line tools (tenant db shell, env test, move db, etc...) it allows us to reduce the boilerplate/repetition of logic in our tools without putting lofic in upstream packages that are used in our various services in prod/staging.
Tools in this package can make assumption that are not safe or optimal for prod/staging environments but are ok to so in the context of command line tools since it improves their usability.
An example for that is `GetTenantByIDOrName` which accepts a string (coming from a command line argument) and tries to determine if the string is a UUID or a name (of a tenant) and returns data about this tenant.
