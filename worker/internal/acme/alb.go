package acme

import (
	"context"
	"crypto/rsa"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmTypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/getsentry/sentry-go"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/acmestorage"
)

func uploadCertToALB(ctx context.Context, tenantID uuid.UUID, ucCert *acmestorage.Certificate, privCertKey *rsa.PrivateKey) error {
	uclog.Infof(ctx, "uploading cert to ACM for tenant %v", tenantID)
	uv := universe.Current()
	if !uv.IsCloud() {
		return nil
	}

	privKey, err := privateKeyToPEM(privCertKey)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// TODO: some day this should only be the regions a given customer is using?
	for _, reg := range region.MachineRegionsForUniverse(uv) {
		uclog.Debugf(ctx, "looking for EB envs in region %v", reg)
		awsCfg, err := ucaws.NewConfigWithRegion(ctx, region.GetAWSRegion(reg))
		if err != nil {
			return ucerr.Wrap(err)
		}

		// upload the cert to ACM
		acmc := acm.NewFromConfig(awsCfg)

		// tag this as a customer cert so we can see later
		customerKey := "CustomerCert"
		customerValue := tenantID.String()
		envKey := "Environment"
		envValue := string(uv)

		ico, err := acmc.ImportCertificate(ctx, &acm.ImportCertificateInput{
			Certificate:      []byte(ucCert.Certificate),
			CertificateChain: []byte(ucCert.CertificateChain),
			PrivateKey:       []byte(privKey),
			Tags:             []acmTypes.Tag{{Key: &customerKey, Value: &customerValue}, {Key: &envKey, Value: &envValue}},
		})
		if err != nil {
			return ucerr.Wrap(err)
		}
		// TODO: in EKS, we need to add the cert ARN to an Ingress resource annotation so it will get picked up by the ALB controller and used with the ALB
		// It is best to use a dedicated Ingress rather than the regular ingress, since the regular ingress is managed by the helm chart/ArgoCD so any modification will be overwritten.
		uclog.Errorf(ctx, "Not implemented for EKS configuration so nothing would be done with the cert ARN %v", *ico.CertificateArn)
		sentry.CaptureMessage("Not implemented for EKS configuration so nothing would be done with the cert ARN")
	}

	return nil
}
