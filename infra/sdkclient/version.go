package sdkclient

import "userclouds.com/infra/service"

// NB: this file is different in the actual SDK repo
// when we're using it in our services, we use a build hash,
// but when we ship a new version of the SDK, we actually
// put a normal x.y.z version number here
var sdkVersion = service.GetBuildHash()
