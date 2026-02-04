package secret

import (
	"context"
	"encoding/json"
	"testing"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
	"userclouds.com/infra/secret/provider/aws"
	"userclouds.com/infra/secret/provider/dev"
	"userclouds.com/infra/secret/provider/kubernetes"
)

// TODO: This represents Config which doesn't use a pointer to `String`.
// Previously we were using a pointer in the test struct and it missed
// an error that was introduced by adding pointer receivers to the
// text Marshaler and sql functions. Should see if we can unpack this
// and get everything to use the pointer receivers.
type testStruct struct {
	Secret String `yaml:"secret" json:"secret" db:"secret"`
}

func TestStringYAML(t *testing.T) {
	ctx := context.Background()
	y := "secret: dev-literal://not-actually-secret"
	var got testStruct
	assert.NoError(t, yaml.Unmarshal([]byte(y), &got))
	s, err := got.Secret.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, s, "not-actually-secret")
	assert.Equal(t, got.Secret.String(), "*********************************") // NB: .String() is required for types to match in assert

	y = "secret: dev://Zm9v"
	got = testStruct{}
	assert.NoError(t, yaml.Unmarshal([]byte(y), &got))
	s, err = got.Secret.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, s, "foo")
}

func TestStringJSON(t *testing.T) {
	ctx := context.Background()
	j := `{"secret":"dev-literal://testme"}`
	var got testStruct
	assert.NoError(t, json.Unmarshal([]byte(j), &got))
	s, err := got.Secret.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, s, "testme")
	assert.Equal(t, got.Secret.String(), "********************") // NB: .String() is required for types to match in assert

	j = `{"secret":"dev://Zm9v"}`
	got = testStruct{}
	assert.NoError(t, json.Unmarshal([]byte(j), &got))
	s, err = got.Secret.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, s, "foo")
	assert.Equal(t, got.Secret.String(), "**********") // NB: .String() is required for types to match in assert

	// make sure we only round trip the location
	bs, err := json.Marshal(got)
	assert.NoError(t, err)
	assert.Equal(t, string(bs), j)
}

func TestJsonMarshal(t *testing.T) {
	s := testStruct{
		Secret: String{
			location: "dev-literal://supersecret",
		},
	}

	expected := `{"secret":"dev-literal://supersecret"}`

	bs, err := json.Marshal(s)
	assert.NoError(t, err)
	assert.Equal(t, expected, string(bs))
}

func TestJsonUnmarshal(t *testing.T) {
	s := `{"secret":"dev-literal://supersecret"}`

	got := testStruct{}
	err := json.Unmarshal([]byte(s), &got)
	assert.NoError(t, err)
	assert.Equal(t, "dev-literal://supersecret", got.Secret.location)
}

func TestFromLocation(t *testing.T) {
	ctx := context.Background()
	devSecret := FromLocation("dev-literal://festivus")
	sv, err := devSecret.Resolve(ctx)
	assert.NoError(t, err)
	assert.Equal(t, sv, "festivus")
	assert.Equal(t, devSecret.String(), "**********************") // NB: .String() is required for types to match in assert
}

func TestString_Value(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{"testsecret", "testsecret", "testsecret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &String{location: tt.input}
			val, err := s.Value()
			assert.NoError(t, err)
			assert.Equal(t, val, tt.output)
		})
	}
}

func TestString_Scan(t *testing.T) {
	tests := []struct {
		name  string
		input any
		valid bool
	}{
		{"string", "testsecret", true},
		{"int", 1, false},
		{"bool", true, false},
		{"complex", struct{ name string }{"name"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &String{}
			err := s.Scan(tt.input)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestString_Validate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"testsecret", "testsecret", true},
		{"empty", "", true},
		{"aws://secrets/testsecret", "aws://secrets/testsecret", true},
		{"kube://secrets/testsecret", "kube://secrets/testsecret", true},
		{"aws://not-a-secret/testsecret", "aws://not-a-secret/testsecret", false},
		// Valid query params
		{"aws with valid profile param", "aws://secrets/testsecret?profile=production", true},
		{"aws with valid region param", "aws://secrets/testsecret?region=us-west-2", true},
		{"aws with both profile and region params", "aws://secrets/testsecret?profile=prod&region=us-east-1", true},
		// Invalid query params
		{"aws with unsupported param", "aws://secrets/testsecret?invalid=value", false},
		{"aws with mixed valid and invalid params", "aws://secrets/testsecret?profile=prod&invalid=value", false},
		{"dev with query params", "dev://testsecret?profile=prod", false},
		{"env with query params", "env://MY_SECRET?profile=prod", false},
		{"kube with query params", "kube://secrets/testsecret?namespace=default", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &String{
				location: tt.input,
			}
			if tt.valid {
				assert.NoError(t, s.Validate())
			} else {
				assert.Error(t, s.Validate())
			}
		})
	}
}

func TestString_NewString_Dev(t *testing.T) {
	tests := []struct {
		description string
		service     string
		name        string
		value       string
		location    string
	}{
		{"simple", "service", "my-secret", "testsecret", "dev://dGVzdHNlY3JldA=="},
	}

	ctx := context.Background()
	t.Setenv("UC_UNIVERSE", "test")

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			s, err := NewStringWithProvider(ctx, tt.service, tt.name, tt.value, dev.New())
			assert.NoError(t, err)
			assert.Equal(t, tt.location, s.location)
		})
	}
}

func TestString_NewString_Kubernetes(t *testing.T) {
	tests := []struct {
		description string
		service     string
		name        string
		value       string
		location    string
		secret      string
		fixture     *corev1.Secret
	}{
		{"simple creation", "service", "my-secret", "testsecret", "kube://secrets/userclouds/test/service/my-secret", "userclouds.test.service.my-secret", &corev1.Secret{}},
		{"simple update", "service", "my-secret", "testsecret", "kube://secrets/userclouds/test/service/my-secret", "userclouds.test.service.my-secret", &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: kubernetes.DefaultNamespace,
			},
			Data: map[string][]byte{
				"value": []byte("secrettoupdate"),
			},
		}},
	}

	ctx := context.Background()
	t.Setenv("UC_UNIVERSE", "test")

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var c *fake.Clientset
			if tt.fixture != nil {
				c = fake.NewSimpleClientset(tt.fixture)
			} else {
				c = fake.NewSimpleClientset()
			}
			s, err := NewStringWithProvider(ctx, tt.service, tt.name, tt.value, kubernetes.New().WithClient(c))
			assert.NoError(t, err)
			assert.Equal(t, tt.location, s.location)

			secret, err := c.CoreV1().Secrets(kubernetes.DefaultNamespace).Get(ctx, tt.secret, metav1.GetOptions{})
			assert.NoError(t, err)
			assert.Equal(t, tt.value, string(secret.Data["value"]))
		})
	}
}

func TestString_Resolve(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
		cached bool
		setup  func(*String)
	}{
		{"empty", "", "", false, func(*String) {}},
		{"no prefix", "testsecret", "testsecret", false, func(*String) {}},
		{"dev", "dev://bXktc2VjcmV0", "my-secret", true, func(*String) {}},
		{"dev-literal", "dev-literal://my-secret", "my-secret", true, func(*String) {}},
		{"env", "env://MY_SECRET", "my-secret", true, func(*String) {
			t.Setenv("MY_SECRET", "my-secret")
		}},
		{"kubernetes", "kube://secrets/my-secret", "testsecret", true, func(s *String) {
			s.WithProvider(kubernetes.New().WithClient(fake.NewSimpleClientset(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: kubernetes.DefaultNamespace,
				},
				Data: map[string][]byte{
					"value": []byte("testsecret"),
				},
			})))
		}},
		{"aws", "aws://secrets/my-secret", "testsecret", true, func(s *String) {
			sm := &aws.MockSecretsManagerClient{}
			input := &secretsmanager.GetSecretValueInput{
				SecretId:     ptr.To("my-secret"),
				VersionStage: ptr.To("AWSCURRENT"),
			}
			sm.On("GetSecretValue", mock.Anything, input, mock.Anything).Return(&secretsmanager.GetSecretValueOutput{
				SecretString: awsv2.String(`{"string":"testsecret"}`),
			}, nil)
			s.WithProvider(aws.New().WithSecretsManagerClient(sm))
		}},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := FromLocation(tt.input)

			tt.setup(s)
			secret, err := s.Resolve(ctx)
			assert.NoError(t, err)
			assert.Equal(t, tt.output, secret)

			if tt.cached {
				cachedValue, found := c.Get(tt.input)
				assert.True(t, found)
				assert.Equal(t, tt.output, cachedValue)
			}
		})
	}
}
