package objectset_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rancher/fleet/internal/cmd/agent/deployer/objectset"
)

func TestObjectByKey_Namespaces(t *testing.T) {
	tests := []struct {
		name           string
		objects        objectset.ObjectByKey
		wantNamespaces []string
	}{
		{
			name:           "empty",
			objects:        objectset.ObjectByKey{},
			wantNamespaces: nil,
		},
		{
			name: "1 namespace",
			objects: objectset.ObjectByKey{
				objectset.ObjectKey{Namespace: "ns1", Name: "a"}: nil,
				objectset.ObjectKey{Namespace: "ns1", Name: "b"}: nil,
			},
			wantNamespaces: []string{"ns1"},
		},
		{
			name: "many namespaces",
			objects: objectset.ObjectByKey{
				objectset.ObjectKey{Namespace: "ns1", Name: "a"}: nil,
				objectset.ObjectKey{Namespace: "ns2", Name: "b"}: nil,
			},
			wantNamespaces: []string{"ns1", "ns2"},
		},
		{
			name: "many namespaces with duplicates",
			objects: objectset.ObjectByKey{
				objectset.ObjectKey{Namespace: "ns1", Name: "a"}: nil,
				objectset.ObjectKey{Namespace: "ns2", Name: "b"}: nil,
				objectset.ObjectKey{Namespace: "ns1", Name: "c"}: nil,
			},
			wantNamespaces: []string{"ns1", "ns2"},
		},
		{
			name: "missing namespace",
			objects: objectset.ObjectByKey{
				objectset.ObjectKey{Namespace: "ns1", Name: "a"}: nil,
				objectset.ObjectKey{Name: "b"}:                   nil,
			},
			wantNamespaces: []string{"", "ns1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNamespaces := tt.objects.Namespaces()
			assert.ElementsMatchf(t, tt.wantNamespaces, gotNamespaces, "Namespaces() = %v, want %v", gotNamespaces, tt.wantNamespaces)
		})
	}
}
