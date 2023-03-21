package conversion

import (
	"reflect"
	"testing"

	v1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	v2 "github.com/maistra/istio-operator/pkg/apis/maistra/v2"
	"github.com/maistra/istio-operator/pkg/controller/versions"
)

var (
	extensionProvidersTestCases []conversionTestCase
	expectedCompleteIstio       = v1.NewHelmValues(map[string]interface{}{
		"global": map[string]interface{}{
			"multiCluster":  globalMultiClusterDefaults,
			"meshExpansion": globalMeshExpansionDefaults,
		},
	})
)

func init() {
	for _, v := range versions.AllV2Versions {
		if v.AtLeast(versions.V2_4) {
			extensionProvidersTestCases = append(extensionProvidersTestCases, extensionProvidersTestCasesV2(v)...)
		}
	}
}

func TestExtensionProvidersConversionFromV2(t *testing.T) {
	for _, tc := range extensionProvidersTestCases {
		t.Run(tc.name, func(t *testing.T) {
			specCopy := tc.spec.DeepCopy()
			helmValues := v1.NewHelmValues(make(map[string]interface{}))
			if err := populateExtensionProvidersValues(specCopy, helmValues.GetContent()); err != nil {
				t.Fatalf("error converting to values: %s", err)
			}
			if !reflect.DeepEqual(tc.isolatedIstio.DeepCopy(), helmValues.DeepCopy()) {
				t.Errorf("unexpected output converting v2 to values:\n\texpected:\n%#v\n\tgot:\n%#v", tc.isolatedIstio.GetContent(), helmValues.GetContent())
			}
			specv2 := &v2.ControlPlaneSpec{}
			// use expected values
			helmValues = tc.isolatedIstio.DeepCopy()
			mergeMaps(tc.completeIstio.DeepCopy().GetContent(), helmValues.GetContent())
			if err := populateExtensionProvidersConfig(helmValues.DeepCopy(), specv2); err != nil {
				t.Fatalf("error converting from values: %s", err)
			}
			assertEquals(t, tc.spec.ExtensionProviders, specv2.ExtensionProviders)
		})
	}
}

func extensionProvidersTestCasesV2(version versions.Version) []conversionTestCase {
	ver := version.String()
	return []conversionTestCase{
		{
			name: "nil." + ver,
			spec: &v2.ControlPlaneSpec{
				Version: ver,
			},
			isolatedIstio: v1.NewHelmValues(map[string]interface{}{}),
			completeIstio: expectedCompleteIstio,
		},
		{
			name: "empty." + ver,
			spec: &v2.ControlPlaneSpec{
				Version:            ver,
				ExtensionProviders: []*v2.ExtensionProviderConfig{},
			},
			isolatedIstio: v1.NewHelmValues(map[string]interface{}{
				"meshConfig": map[string]interface{}{
					"extensionProviders": []interface{}{},
				},
			}),
			completeIstio: expectedCompleteIstio,
		},
		{
			name: "prometheus." + ver,
			spec: &v2.ControlPlaneSpec{
				Version: ver,
				ExtensionProviders: []*v2.ExtensionProviderConfig{
					{
						Name:       "prometheus",
						Prometheus: &v2.ExtensionProviderPrometheusConfig{},
					},
				},
			},
			isolatedIstio: v1.NewHelmValues(map[string]interface{}{
				"meshConfig": map[string]interface{}{
					"extensionProviders": []interface{}{
						map[string]interface{}{
							"name":       "prometheus",
							"prometheus": map[string]interface{}{},
						},
					},
				},
			}),
			completeIstio: expectedCompleteIstio,
		},
		{
			name: "envoyExtAuthzHttp." + ver,
			spec: &v2.ControlPlaneSpec{
				Version: ver,
				ExtensionProviders: []*v2.ExtensionProviderConfig{
					{
						Name: "ext-authz-http",
						EnvoyExtAuthzHttp: &v2.ExtensionProviderEnvoyExternalAuthorizationHttpConfig{
							Service: "ext-authz.foo.svc.cluster.local",
							Port:    8000,
						},
					},
				},
			},
			isolatedIstio: v1.NewHelmValues(map[string]interface{}{
				"meshConfig": map[string]interface{}{
					"extensionProviders": []interface{}{
						map[string]interface{}{
							"name": "ext-authz-http",
							"envoyExtAuthzHttp": map[string]interface{}{
								"service": "ext-authz.foo.svc.cluster.local",
								"port":    8000,
							},
						},
					},
				},
			}),
			completeIstio: expectedCompleteIstio,
		},
		{
			name: "prometheus-and-envoyExtAuthzHttp." + ver,
			spec: &v2.ControlPlaneSpec{
				Version: ver,
				ExtensionProviders: []*v2.ExtensionProviderConfig{
					{
						Name:       "prometheus",
						Prometheus: &v2.ExtensionProviderPrometheusConfig{},
					},
					{
						Name: "ext-authz-http",
						EnvoyExtAuthzHttp: &v2.ExtensionProviderEnvoyExternalAuthorizationHttpConfig{
							Service: "ext-authz.foo.svc.cluster.local",
							Port:    8000,
						},
					},
				},
			},
			isolatedIstio: v1.NewHelmValues(map[string]interface{}{
				"meshConfig": map[string]interface{}{
					"extensionProviders": []interface{}{
						map[string]interface{}{
							"name":       "prometheus",
							"prometheus": map[string]interface{}{},
						},
						map[string]interface{}{
							"name": "ext-authz-http",
							"envoyExtAuthzHttp": map[string]interface{}{
								"service": "ext-authz.foo.svc.cluster.local",
								"port":    8000,
							},
						},
					},
				},
			}),
			completeIstio: expectedCompleteIstio,
		},
	}
}
