package uckube

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"userclouds.com/infra/uclog"
)

// GetSecret retrieves a secret and returns the value.
func GetSecret(ctx context.Context, client kubernetes.Interface, name string, namespace string) (string, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if value, ok := secret.Data["value"]; ok {
		return string(value), nil
	}

	return "", fmt.Errorf("secret does not contain value field")
}

// CreateOrUpdateSecret checks for the existence of a secret and then creates or
// updates the value.
func CreateOrUpdateSecret(ctx context.Context, client kubernetes.Interface, name string, namespace string, value string) error {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			uclog.Debugf(ctx, "Creating secret %s/%s", namespace, name)
			s := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "userclouds",
					},
				},
				Data: map[string][]byte{
					"value": []byte(value),
				},
			}
			_, err := client.CoreV1().Secrets(namespace).Create(ctx, s, metav1.CreateOptions{})
			return err
		}
		return err
	}

	secret.Data = map[string][]byte{
		"value": []byte(value),
	}
	secret, err = client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if string(secret.Data["value"]) != value {
		return fmt.Errorf("secret does not contain expected value")
	}

	return nil
}

// DeleteSecret removes a secret if it exists.
func DeleteSecret(ctx context.Context, client kubernetes.Interface, name string, namespace string) error {
	err := client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		uclog.Debugf(ctx, "Secret %s/%s not found", namespace, name)
		return nil
	}

	return err
}
