package k8sutils

import (
	"fmt"
	"testing"

	"github.com/dell/csm-metrics-powermax/internal/reverseproxy/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetCertFileFromSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret *corev1.Secret
		want   error
	}{
		{
			name: "valid secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
				},
				Data: map[string][]byte{
					"cert": []byte("test-cert"),
				},
			},
		},
		{
			name:   "Empty Secret",
			secret: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			k8sUtils = &K8sUtils{
				CertDirectory: "/test/certs",
			}
			defer func() { k8sUtils = nil }()
			utils := k8sUtils

			_, _ = utils.getCertFileFromSecret(tt.secret)
		})
	}
}

func TestGetCredentialFromSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  *corev1.Secret
		want    *common.Credentials
		wantErr error
	}{
		{
			name: "valid secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "Test-Secret",
				},
				Data: map[string][]byte{
					"username": []byte("test-username"),
					"password": []byte("test-password"),
				},
			},
			want: &common.Credentials{
				UserName: "test-username",
				Password: "test-password",
			},
			wantErr: nil,
		},
		{
			name:    "Empty Secret",
			secret:  nil,
			want:    nil,
			wantErr: fmt.Errorf("secret can't be nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utils := &K8sUtils{
				CertDirectory: "/test/certs",
			}
			k8sUtils = utils
			defer func() { k8sUtils = nil }()

			got, err := utils.getCredentialFromSecret(tt.secret)
			assert.Equal(t, err, tt.wantErr)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestGetSecret(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		secretName string
		setup      func() (*KubernetesClient, error)
		want       *corev1.Secret
		wantErr    error
	}{
		{
			name:       "valid secret",
			namespace:  "test-namespace",
			secretName: "test-secret",
			setup: func() (*KubernetesClient, error) {
				client := fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"key": []byte("value"),
					},
				})
				return &KubernetesClient{Clientset: client}, nil
			},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.setup()
			if err != nil {
				t.Fatalf("failed to setup client: %s", err.Error())
			}

			secret, err := client.GetSecret(tt.namespace, tt.secretName)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Equal(t, tt.want, secret)
			}
		})
	}
}

func TestGetCertFileFromSecretName(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		secretName string
		setup      func() (*K8sUtils, error)
		wantErr    error
	}{
		{
			name:       "valid secret",
			namespace:  "test-namespace",
			secretName: "test-secret",
			setup: func() (*K8sUtils, error) {
				client := fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"cert": []byte("file-name"),
					},
				})
				return &K8sUtils{
					KubernetesClient: &KubernetesClient{Clientset: client},
					Namespace:        "test-namespace",
				}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.setup()
			if err != nil {
				t.Fatalf("failed to setup client: %s", err.Error())
			}
			k8sUtils = client
			defer func() { k8sUtils = nil }()

			fileName, err := client.GetCertFileFromSecretName(tt.secretName)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NotEmpty(t, fileName)
			}
		})
	}
}

func TestGetCredentialsFromSecretName(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		secretName string
		setup      func() (*K8sUtils, error)
		wantErr    error
	}{
		{
			name:       "valid secret",
			namespace:  "test-namespace",
			secretName: "test-secret",
			setup: func() (*K8sUtils, error) {
				client := fake.NewSimpleClientset(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-namespace",
					},
					Data: map[string][]byte{
						"username": []byte("user"),
						"password": []byte("pass"),
					},
				})
				return &K8sUtils{
					KubernetesClient: &KubernetesClient{Clientset: client},
					Namespace:        "test-namespace",
				}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.setup()
			if err != nil {
				t.Fatalf("failed to setup client: %s", err.Error())
			}
			k8sUtils = client
			defer func() { k8sUtils = nil }()

			credentials, err := client.GetCredentialsFromSecretName(tt.secretName)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.Equal(t, "user", credentials.UserName)
				assert.Equal(t, "pass", credentials.Password)
			}
		})
	}
}

// func TestInit(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		namespace  string
// 		secretName string
// 		setup      func() (*K8sUtils, error)
// 		wantErr    error
// 	}{
// 		{
// 			name:       "valid secret",
// 			namespace:  "test-namespace",
// 			secretName: "test-secret",
// 			setup: func() (*K8sUtils, error) {
// 				client := fake.NewSimpleClientset(&corev1.Secret{
// 					ObjectMeta: metav1.ObjectMeta{
// 						Name:      "test-secret",
// 						Namespace: "test-namespace",
// 					},
// 					Data: map[string][]byte{
// 						"username": []byte("user"),
// 						"password": []byte("pass"),
// 					},
// 				})
// 				return &K8sUtils{
// 					KubernetesClient: &KubernetesClient{Clientset: client},
// 					Namespace:        "test-namespace",
// 				}, nil
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// client, err := tt.setup()
// 			// if err != nil {
// 			// 	t.Fatalf("failed to setup client: %s", err.Error())
// 			// }

// 			utils, err := Init(tt.namespace, "", false, 0)
// 			if err != nil {
// 				assert.EqualError(t, err, tt.wantErr.Error())
// 			} else {
// 				assert.NotNil(t, utils.KubernetesClient)
// 				// assert.Equal(t, "user", credentials.UserName)
// 				// assert.Equal(t, "pass", credentials.Password)
// 			}
// 		})
// 	}
// }
