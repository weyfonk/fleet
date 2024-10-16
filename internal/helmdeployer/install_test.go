package helmdeployer_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/fleet/internal/helmdeployer"
)

func TestValuesFrom(t *testing.T) {
	a := assert.New(t)
	key := "values.yaml"
	newline := "\n"
	if runtime.GOOS == "windows" {
		newline = "\r\n"
	}

	configMapPayload := fmt.Sprintf("replication: \"true\"%sreplicas: \"2\"%sserviceType: NodePort", newline, newline)
	secretPayload := fmt.Sprintf("replication: \"false\"%sreplicas: \"3\"%sserviceType: NodePort%sfoo: bar", newline, newline, newline)
	totalValues := map[string]interface{}{"beforeMerge": "value"}
	expected := map[string]interface{}{
		"beforeMerge": "value",
		"replicas":    "2",
		"replication": "true",
		"serviceType": "NodePort",
		"foo":         "bar",
	}

	configMapName := "configmap-name"
	configMapNamespace := "configmap-namespace"
	configMapValues, err := helmdeployer.ValuesFromConfigMap(configMapName, configMapNamespace, key, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: configMapNamespace,
		},
		Data: map[string]string{
			key: configMapPayload,
		},
	})
	a.NoError(err)

	secretName := "secret-name"
	secretNamespace := "secret-namespace"
	secretValues, err := helmdeployer.ValuesFromSecret(secretName, secretNamespace, key, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
		Data: map[string][]byte{
			key: []byte(secretPayload),
		},
	})
	a.NoError(err)

	totalValues = helmdeployer.MergeValues(totalValues, secretValues)
	totalValues = helmdeployer.MergeValues(totalValues, configMapValues)
	a.Equal(expected, totalValues)
}
