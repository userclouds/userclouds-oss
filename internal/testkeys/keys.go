package testkeys

import (
	"crypto/rsa"
	"testing"

	"userclouds.com/infra/assert"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucjwt"
	"userclouds.com/internal/tenantplex"
)

const privateKeyString = `-----BEGIN RSA PRIVATE KEY-----
MIIG4wIBAAKCAYEAuwrugnd/mqeKStVM6HB3Ju1Hfg/KFHqrxS1JP2xHsglwXHyn
jX5Q6TlLexMfIZEVyR47Nde1lPz4y5Cu72yVSeGnf3kOM11ytlowjO+DQzLngN0U
FP66ZcUUduqJYRKuHPEBu9G9rwr4pEHBHRYwOr/B8Ok96z+YTIZd2FQe/UYptdda
x8QKfPmmQA4jzjZRCSZYPl9BaTjz00C2jRop8BD1sEBWKrECOZ6YHpyiLzY9c5I4
UQBgV1GSdokJFe2OCBFzst7NV3n+ek1CyCaWDUw82Pp/sPv0bFxwmM3xBgkCH266
YDh/bIalq+SvJlXdwrMl1r3IIA9QKAIpZv0RDR45xkDrWfRNw9EKCB4s97jCp92j
2bOUpMt/CnSFNnqNxfocJKN/VqNXb712A5AYX84ShGh3CriEHrFhx//WfBFb3spG
jNHu9ekfFruK+Lia09k3738e3gB9bhZRLstG/suNniYtPZfL95zU+cTK9MZhud7g
QtZhkusOs4ifUWTlAgMBAAECggGAN0MUkwGBdw0XI+L/dRF9csfaPpmlqAVSaNBn
etCgIi79vqWpz3lJqI6gCX3tzboTCLfg4JiZ8qoHTAW0WdLoDMsZ9OSsWGq8sLnW
7Fz7mEga9AzdmRJluhnPYQ8MhdzCCpT+YSKn+2avbcBrsQ9UMpdjUq1m+PFyKvHs
GjVIbqZjPnGhRbJbMu+DuhszYwLTUHO+0LbOGauVBo5xISFg0KgCHw/zJWvk72c3
JJw8otxQrau+7dfBnyrfrhvwzkTAF0yhqbYMJrXkiEZqp46qGrQxRCGKyTdSjbh2
tAm7mNryMYuVu8tg0AD50UEV0B9+wRB8t3W9LMEL2fjASHMfQteqkKHyw/4d6Qtx
TBqAO6X6gY+viH9A0k9NrBWPUmY/tDnyt7payzNLWMFnhgEIyHHr/3MHqP7ulq/B
SBLwFChshKut63wDhTn+XzucjkOobDvj+2EY1obpZPaBZgDY2F5zsC+8SW3+qpdj
WO714fhcp+PCHW491g8KcTgYc+ihAoHBAO238nXLaD8w3HvChq0KvRhPYdaICaPJ
o6wloyvd2nFrl60X15tedOTdp5pK1X3SCuUc6ZSokxBi/eyjRl1pky4Hyrs5xef+
proMc1Z+DKA6eu18dn8PKzJWGI9FE13FD9Y65sAsh8yfvKlaWdgFOpkLYAbZFHqk
3hBsze8HhVPLygmilK96cbPHkMDrwnak7IvDFF0DpXKkAtADfU7dPthUB26020ny
85u4FtszMeEjqgafLxsPiwbFaHJrkwDjPQKBwQDJbU9/DYbMcuvkMWaa/f+tO6Ew
f6uMGrgFLNCeUBdkeEAsHKSqGp4+f1XNBkMlh429AcPyaBGVH7hZ9M+InLyvcJU/
gkTfWjoD0FMPjQESaQgnZe99e4300SDkuerEZNDCp4RhKJJ7pD8xGQ4Vj8YvA0Ng
ISwrv023qkmt8hZ+etd657/xYsROJ805lbgQDiv2dI+mq32MoPp+WCC44nqF5/E9
xMq2BEYSs2piz9vHLMAvT19GUA5nEEOpRnzCgskCgcBmfNr0vCiKrecxGFH2At45
v+e/lVEKo2GEU4nA3NpT8f4nq1LScmvVTFb5J3BZ2ZfG5asy42bcNsGhJ1er6FuD
Cer2w1a2ycxaBAop7RhGcFAVWYbBCuolvobCJhbOY6qLQ0O+8LPvnaK6JPD9OGvm
Fchly2uP4Mq4rCAxAL4TvZWyh7yw1wp0ZwLamgpyGnK9YvLBk1PeVCW+RvLccHiq
zbeSnDi67hrnNPvtr2m+1iB00GZ/tTjMR4nbYtOzG0UCgcBxDhmAhmcSea5M9i1Q
8R+Aa+edAQuYJ6cBwJWXRfzbN2NNXwZNM4N+MJpH6Svm9J5pZ4RDmoXD3Xnrg6y4
UMDW96nNa6CcfFfzrAnywIHJg4pAEsbI94BF2NtNhcxvTuadWsjCf7M4EoglVprB
H2FtIbe/TN8t7sIARGP2bdqSQwCOy2TAZ18nPs/BcndNC6dBPUsjkT12oSP3ph83
pmZ+oiCVOs9MOjnaZTlhHKmOsV9tLm+bV3O+BTL038tGoYECgcEA4rkkSyq0zY/e
fqAw04cNmOdS8a/0qvnMhVh/jjmWS0evxenVODPuJR2uuEcIk1bQz2z0nckAS2rh
3euLqrYhc8f52HpRGZ4HDxbE6o7gA0jNroUvLqDnnmEnEz6ol64iDJvCwC/ytuZy
cT+1v48OQp4dktcAWxa43gcj9rjc8FB1fdmgmBCNZPu8W1+ZlOqt0iwAVB5RKyBs
DDqbA5hqNbhrCwC0Jer7N06w9BQ5/m4oFK+jbWWYt7j2siYVjrR9
-----END RSA PRIVATE KEY-----`

const publicKeyString = `-----BEGIN PUBLIC KEY-----
MIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAuwrugnd/mqeKStVM6HB3
Ju1Hfg/KFHqrxS1JP2xHsglwXHynjX5Q6TlLexMfIZEVyR47Nde1lPz4y5Cu72yV
SeGnf3kOM11ytlowjO+DQzLngN0UFP66ZcUUduqJYRKuHPEBu9G9rwr4pEHBHRYw
Or/B8Ok96z+YTIZd2FQe/UYptddax8QKfPmmQA4jzjZRCSZYPl9BaTjz00C2jRop
8BD1sEBWKrECOZ6YHpyiLzY9c5I4UQBgV1GSdokJFe2OCBFzst7NV3n+ek1CyCaW
DUw82Pp/sPv0bFxwmM3xBgkCH266YDh/bIalq+SvJlXdwrMl1r3IIA9QKAIpZv0R
DR45xkDrWfRNw9EKCB4s97jCp92j2bOUpMt/CnSFNnqNxfocJKN/VqNXb712A5AY
X84ShGh3CriEHrFhx//WfBFb3spGjNHu9ekfFruK+Lia09k3738e3gB9bhZRLstG
/suNniYtPZfL95zU+cTK9MZhud7gQtZhkusOs4ifUWTlAgMBAAE=
-----END PUBLIC KEY-----`

// Config will validate but is useless
// NB: these are test-only keys randomly generated -- don't use them elsewhere :)
var Config = tenantplex.Keys{
	KeyID:      "deadbeef",
	PrivateKey: secret.NewTestString(privateKeyString),
	PublicKey:  publicKeyString,
}

var cachedPrivateKey *rsa.PrivateKey
var cachedPublicKey *rsa.PublicKey

// GetPrivateKey is an *rsa.PrivateKey for tests
func GetPrivateKey(t *testing.T) *rsa.PrivateKey {
	if cachedPrivateKey != nil {
		return cachedPrivateKey
	}
	pk, err := ucjwt.LoadRSAPrivateKey([]byte(privateKeyString))
	assert.NoErr(t, err)
	return pk
}

// GetPublicKey is an *rsa.PublicKey for tests
func GetPublicKey() (*rsa.PublicKey, error) {
	if cachedPublicKey != nil {
		return cachedPublicKey, nil
	}
	pk, err := ucjwt.LoadRSAPublicKey([]byte(publicKeyString))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return pk, nil
}
