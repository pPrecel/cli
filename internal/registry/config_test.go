package registry

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetConfig(t *testing.T) {
	t.Run("Should return the RegistryConfig", func(t *testing.T) {
		// given
		client := fake.NewSimpleClientset(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      RegistrySecretName,
				Namespace: serverlessNamespace,
			},
			Data: map[string][]byte{
				".dockerconfigjson": []byte(`{"auths": {}}`),
				"username":          []byte("testUsername"),
				"password":          []byte("testPassword"),
				"pullRegAddr":       []byte("pullRegAddr"),
				"pushRegAddr":       []byte("pushRegAddr"),
				"isInternal":        []byte("true"),
			},
		})
		expectedRegistryConfig := &RegistryConfig{
			DockerConfigJson: `{"auths": {}}`,
			Username:         "testUsername",
			Password:         "testPassword",
			PullRegAddr:      "pullRegAddr",
			PushRegAddr:      "pushRegAddr",
			IsInternal:       true,
		}

		// when
		config, err := GetConfig(context.Background(), client)

		// then
		require.NoError(t, err)
		require.Equal(t, expectedRegistryConfig, config)
	})

	t.Run("Should return an error when the secret does not exist", func(t *testing.T) {
		// given
		client := fake.NewSimpleClientset()
		expectedErrorMsg := "secrets \"serverless-registry-config-default\" not found"

		// when
		config, err := GetConfig(context.Background(), client)

		// then
		require.Error(t, err)
		require.Nil(t, config)
		require.Contains(t, err.Error(), expectedErrorMsg)
	})
}

func TestGetWorkloadMeta(t *testing.T) {
	t.Run("Should return the RegistryPodMeta", func(t *testing.T) {
		// given
		client := fake.NewSimpleClientset(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "serverless-docker-registry",
				Namespace: "kyma-system",
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "serverless-docker-registry",
				},
				Ports: []corev1.ServicePort{
					{
						TargetPort: intstr.FromString("5000"),
					},
				},
			},
		}, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "serverless-docker-registry-7d4d7b7b4f-7z5zv",
				Namespace: "kyma-system",
				Labels: map[string]string{
					"app": "serverless-docker-registry",
				},
			},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					{
						Type:   corev1.ContainersReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		})
		config := &RegistryConfig{
			PushRegAddr: "serverless-docker-registry.kyma-system.svc.cluster.local:5000",
		}
		expectedRegistryPodMeta := &RegistryPodMeta{
			Name:      "serverless-docker-registry-7d4d7b7b4f-7z5zv",
			Namespace: "kyma-system",
			Port:      "5000",
		}

		// when
		meta, err := GetWorkloadMeta(context.Background(), client, config)

		// then
		require.NoError(t, err)
		require.Equal(t, expectedRegistryPodMeta, meta)
	})
	t.Run("Should return an error when no pods exist", func(t *testing.T) {
		// given
		client := fake.NewSimpleClientset(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "serverless-docker-registry",
				Namespace: "kyma-system",
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app": "serverless-docker-registry",
				},
				Ports: []corev1.ServicePort{
					{
						TargetPort: intstr.FromString("5000"),
					},
				},
			},
		})
		config := &RegistryConfig{
			PushRegAddr: "serverless-docker-registry.kyma-system.svc.cluster.local:5000",
		}

		// when
		meta, err := GetWorkloadMeta(context.Background(), client, config)

		// then
		require.Error(t, err)
		require.Nil(t, meta)
		require.Contains(t, err.Error(), "no ready registry pod found")
	})
}