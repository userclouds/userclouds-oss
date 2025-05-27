# UserClouds SDK - Golang

[![Go Reference](https://pkg.go.dev/badge/userclouds.com)](https://pkg.go.dev/userclouds.com)

Changelog is available [here](./CHANGELOG.md)

This repo contains the client SDK for UserClouds AuthZ & IDP services as well as a sample app to demonstrate use of the AuthZ service.

## Golang Sample

To run the sample from the root of the repo:

1. `cd samples/basic`
2. `cp .env.example .env`
3. Open `.env` in a text editor of your choice and fill in the Tenant URL, Client ID, and Client Secret from the UserClouds console. You can find this information in the `Plex` settings page (you may need to expand the `Plex Apps` section for the Client ID & Secret)
4. `go run *.go` -> this runs a command line app which talks to your UserClouds tenant, creates test users & AuthZ objects/ACLs, and runs through some test scenarios. At the end, it will display a sample file/directory tree with permissions on each node of the tree.
