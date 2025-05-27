package main

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

type secretCopier struct {
	sourceClient  *secretsmanager.Client
	targetClient  *secretsmanager.Client
	dryRun        bool
	overwrite     bool
	sourceAccount string
	targetAccount universe.Universe
	renameSecrets bool
}

func (sc *secretCopier) copySecrets(ctx context.Context, secretsFilter string) (int, error) {
	paginator := sc.getSecretsPaginator(secretsFilter)
	count := 0
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return count, ucerr.Wrap(err)
		}
		for _, secret := range resp.SecretList {
			count++
			err = sc.copySecret(ctx, secret)
			if err != nil {
				return count, ucerr.Wrap(err)
			}
		}
	}
	if count == 0 {
		return count, ucerr.Errorf("no secrets found for filter '%s' in %v", secretsFilter, sc.sourceAccount)
	}
	return count, nil
}

func (sc *secretCopier) getSecretsPaginator(filter string) *secretsmanager.ListSecretsPaginator {
	return secretsmanager.NewListSecretsPaginator(sc.sourceClient, &secretsmanager.ListSecretsInput{
		Filters: []types.Filter{{Key: "name", Values: []string{filter}}},
	})

}

func (sc *secretCopier) getSecretName(secret types.SecretListEntry) (string, error) {
	sn := *secret.Name
	if !sc.renameSecrets {
		return sn, nil
	}
	newName := strings.Replace(sn, string(sc.sourceAccount), string(sc.targetAccount), 1)
	if newName == sn {
		return "", ucerr.Errorf("failed to rename secret %s", sn)
	}
	return newName, nil

}

func (sc *secretCopier) copySecret(ctx context.Context, secret types.SecretListEntry) error {
	// Get a secret from the source account
	dryRunStr := ""
	if sc.dryRun {
		dryRunStr = " [dry run]"
	}
	secretName, err := sc.getSecretName(secret)
	if err != nil {
		return ucerr.Wrap(err)
	}
	resp, err := sc.sourceClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: secret.ARN})
	if err != nil {
		return ucerr.Wrap(err)
	}

	listSecretsResp, err := sc.targetClient.ListSecrets(ctx, &secretsmanager.ListSecretsInput{Filters: []types.Filter{{Key: "name", Values: []string{secretName}}}})
	if err != nil {
		return ucerr.Wrap(err)
	}
	var targetSecret *types.SecretListEntry

	if len(listSecretsResp.SecretList) > 0 {
		// The ListSecrets API may return multiple secrets.
		// For example, if queried for the secret "jerry", it might also return "jerry-seinfeld" and "jerry".
		// Therefore, we need to find the one that matches the name exactly.
		for _, s := range listSecretsResp.SecretList {
			if *s.Name == secretName {
				targetSecret = &s
				break
			}
		}
	}
	if targetSecret != nil {
		if sc.overwrite {
			uclog.Infof(ctx, "Overwriting existing secret: %s%s", secretName, dryRunStr)
			if !sc.dryRun {
				_, err = sc.targetClient.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{SecretId: targetSecret.ARN, SecretString: resp.SecretString})
				if err != nil {
					return ucerr.Wrap(err)
				}
			}
		} else {
			uclog.Infof(ctx, "Skipping existing secret: %s", secretName)
		}
	} else {
		uclog.Infof(ctx, "Creating new secret: %s%s", secretName, dryRunStr)
		if !sc.dryRun {
			_, err = sc.targetClient.CreateSecret(ctx, &secretsmanager.CreateSecretInput{Name: &secretName, SecretString: resp.SecretString})
			if err != nil {
				return ucerr.Wrap(err)
			}
		}
	}
	return nil
}
