package k8sutils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/reverseproxy/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
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

			_, _ = utils.GetCertFileFromSecret(tt.secret)
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
			name: "secret doesn't contain username or password",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "Test-Secret",
				},
				Data: map[string][]byte{},
			},
			want:    nil,
			wantErr: errors.New("username not found in secret data"),
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

			got, err := utils.GetCredentialsFromSecret(tt.secret)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
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

func TestStartInformer(t *testing.T) {
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

				informerFactory := informers.NewSharedInformerFactory(client, time.Second)
				secretInformer := informerFactory.Core().V1().Secrets()

				return &K8sUtils{
					KubernetesClient: &KubernetesClient{Clientset: client},
					Namespace:        "test-namespace",
					SecretInformer:   secretInformer,
					InformerFactory:  informerFactory,
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

			err = client.StartInformer(func(ui UtilsInterface, s *corev1.Secret) {})
			assert.Nil(t, err)

			// TODO: waiting here allows the UpdateFunc callback to be invoked
			time.Sleep(1 * time.Second)

		})
	}
}

func TestCreateOutOfClusterKubeClient(t *testing.T) {
	client := &KubernetesClient{}

	home := os.Getenv("HOME")
	os.Setenv("HOME", "")
	defer os.Setenv("HOME", home)

	wd, _ := os.Getwd()
	str := filepath.Join(wd, "..", "k8s/testdata")
	os.Setenv("X_CSI_KUBECONFIG_PATH", str)

	err := client.CreateOutOfClusterKubeClient()
	assert.Nil(t, err)
	assert.NotNil(t, client.Clientset)
}

func TestInit(t *testing.T) {
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
			home := os.Getenv("HOME")
			os.Setenv("HOME", "")
			defer os.Setenv("HOME", home)

			wd, _ := os.Getwd()
			str := filepath.Join(wd, "..", "k8s/testdata")
			os.Setenv("X_CSI_KUBECONFIG_PATH", str)

			utils, err := Init(tt.namespace, "", false, 0)
			assert.NotNil(t, utils)
			assert.Nil(t, err)
		})
	}
}

func TestCreateInClusterKubeClient(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
	}{
		{
			name:    "not in cluster",
			wantErr: errors.New("unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KubernetesClient{}
			err := c.CreateInClusterKubeClient()
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
