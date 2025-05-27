This package node (infra/namespace) exists as a place to hang things that are useful across our
code and should live at or near the bottom of the dependency stack (eg. the packages in namespace
should not depend on anything else in userclouds.com/). This allows them to be safely used in
widespread packages like `uclog`

_universe_ - what UC environment are we running in? dev, prod, etc. We chose universe because
environment was so common and commonly overloaded, and could also refer to eg. different customer
tenants for their different environments

_region_ - which UC "datacenter" are we running in? Functionally this is usually a concatenation of
"{cloud provider}-{provider datacenter name}" like "aws-us-west-2" although as always they should be
treated as opaque strings (or simply opaque types) outside of the package.
