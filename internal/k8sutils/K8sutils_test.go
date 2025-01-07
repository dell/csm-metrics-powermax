package k8sutils

import (
	"fmt"
	"testing"

	"github.com/dell/csm-metrics-powermax/internal/reverseproxy/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

			got, err := utils.getCredentialFromSecret(tt.secret)
			assert.Equal(t, err, tt.wantErr)
			assert.Equal(t, got, tt.want)
		})
	}
}
