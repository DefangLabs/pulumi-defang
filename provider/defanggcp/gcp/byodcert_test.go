package gcp

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/certificatemanager"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
)

const (
	certificateType         = "gcp:certificatemanager/certificate:Certificate"
	certificateMapType      = "gcp:certificatemanager/certificateMap:CertificateMap"
	certificateMapEntryType = "gcp:certificatemanager/certificateMapEntry:CertificateMapEntry"
	dnsAuthorizationType    = "gcp:certificatemanager/dnsAuthorization:DnsAuthorization"
	dnsRecordSetType        = "gcp:dns/recordSet:RecordSet"
)

var errInjectedByodRegistration = errors.New("injected resource-registration failure")

type byodResource struct {
	name   string
	typ    string
	inputs resource.PropertyMap
}

type byodMocks struct {
	mu               sync.RWMutex
	resources        []byodResource
	failType         string
	failNameContains string
}

func (m *byodMocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	if args.TypeToken == m.failType && strings.Contains(args.Name, m.failNameContains) {
		return "", nil, errInjectedByodRegistration
	}

	outputs := make(resource.PropertyMap, len(args.Inputs)+1)
	for key, value := range args.Inputs {
		outputs[key] = value
	}
	switch args.TypeToken {
	case certificateMapType:
		outputs["name"] = resource.NewStringProperty(args.Name)
	case dnsAuthorizationType:
		domain := args.Inputs["domain"].StringValue()
		outputs["dnsResourceRecords"] = resource.NewArrayProperty([]resource.PropertyValue{
			resource.NewObjectProperty(resource.PropertyMap{
				"name": resource.NewStringProperty("_acme-challenge." + domain + "."),
				"type": resource.NewStringProperty("CNAME"),
				"data": resource.NewStringProperty("challenge." + domain + ".authorize.certificatemanager.goog."),
			}),
		})
	case "gcp:compute/globalAddress:GlobalAddress":
		outputs["address"] = resource.NewStringProperty("203.0.113.10")
	}

	m.mu.Lock()
	m.resources = append(m.resources, byodResource{name: args.Name, typ: args.TypeToken, inputs: args.Inputs})
	m.mu.Unlock()
	return args.Name + "_id", outputs, nil
}

func (m *byodMocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func (m *byodMocks) Resources() []byodResource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]byodResource(nil), m.resources...)
}

func byodService(domain string, aliases ...string) compose.ServiceConfig {
	svc := compose.ServiceConfig{DomainName: domain}
	if aliases != nil {
		svc.Networks = map[compose.NetworkID]compose.ServiceNetworkConfig{
			compose.DefaultNetwork: {Aliases: aliases},
		}
	}
	return svc
}

func runCreateByodCerts(
	mocks *byodMocks,
	entries []LBServiceEntry,
	zones map[string]string,
	withPublicIP bool,
) error {
	return pulumi.RunErr(func(ctx *pulumi.Context) error {
		certMap, err := certificatemanager.NewCertificateMapResource(ctx, "test-cert-map", nil)
		if err != nil {
			return err
		}
		infra := &SharedInfra{DnsZones: zones}
		if withPublicIP {
			infra.PublicIP, err = compute.NewGlobalAddress(ctx, "test-public-ip", nil)
			if err != nil {
				return err
			}
		}
		return createByodCerts(ctx, certMap, infra, entries)
	}, pulumi.WithMocks("project", "stack", mocks))
}

func resourcesOfType(resources []byodResource, typ string) []byodResource {
	var matches []byodResource
	for _, registered := range resources {
		if registered.typ == typ {
			matches = append(matches, registered)
		}
	}
	return matches
}

func recordByName(t *testing.T, resources []byodResource, name string) byodResource {
	t.Helper()
	for _, registered := range resourcesOfType(resources, dnsRecordSetType) {
		if registered.name == name {
			return registered
		}
	}
	require.Failf(t, "missing DNS record", "name=%s resources=%v", name, resources)
	return byodResource{}
}

func propertyStrings(value resource.PropertyValue) []string {
	values := value.ArrayValue()
	result := make([]string, len(values))
	for i, item := range values {
		result[i] = item.StringValue()
	}
	return result
}

func TestCreateByodCertsReusesAuthorizationAndKeepsSeparateHostEntries(t *testing.T) {
	mocks := &byodMocks{}
	entries := []LBServiceEntry{
		{Name: "api", Config: byodService("Example.COM.", "*.example.com")},
		// Both names normalize to requests already made by api. The first service
		// deterministically owns the duplicate map entries.
		{Name: "web", Config: byodService("*.EXAMPLE.COM.", "example.com")},
	}
	zones := map[string]string{
		"example.com":   "trusted-zone",
		"*.example.com": "trusted-zone",
	}
	require.NoError(t, runCreateByodCerts(mocks, entries, zones, true))
	resources := mocks.Resources()

	authorizations := resourcesOfType(resources, dnsAuthorizationType)
	require.Len(t, authorizations, 1)
	require.Equal(t, "example.com", authorizations[0].inputs["domain"].StringValue())
	require.Equal(t, byodResourceName("authz", "example.com"), authorizations[0].name)

	challenge := recordByName(t, resources, byodResourceName("authz", "example.com")+"-record")
	require.Equal(t, "trusted-zone", challenge.inputs["managedZone"].StringValue())
	require.Equal(t, "_acme-challenge.example.com.", challenge.inputs["name"].StringValue())
	require.Equal(t, "CNAME", challenge.inputs["type"].StringValue())
	require.Equal(t, []string{"challenge.example.com.authorize.certificatemanager.goog."},
		propertyStrings(challenge.inputs["rrdatas"]))

	for _, hostname := range []string{"example.com", "*.example.com"} {
		aRecord := recordByName(t, resources, byodResourceName("cert", hostname)+"-record")
		require.Equal(t, "trusted-zone", aRecord.inputs["managedZone"].StringValue())
		require.Equal(t, hostname+".", aRecord.inputs["name"].StringValue())
		require.Equal(t, "A", aRecord.inputs["type"].StringValue())
		require.Equal(t, []string{"203.0.113.10"}, propertyStrings(aRecord.inputs["rrdatas"]))
	}

	certificates := resourcesOfType(resources, certificateType)
	require.Len(t, certificates, 2)
	certificateDomains := make([]string, 0, len(certificates))
	for _, certificate := range certificates {
		managed := certificate.inputs["managed"].ObjectValue()
		certificateDomains = append(certificateDomains, propertyStrings(managed["domains"])...)
		require.Equal(t, []string{byodResourceName("authz", "example.com") + "_id"},
			propertyStrings(managed["dnsAuthorizations"]))
	}
	sort.Strings(certificateDomains)
	require.Equal(t, []string{"*.example.com", "example.com"}, certificateDomains)

	mapEntries := resourcesOfType(resources, certificateMapEntryType)
	require.Len(t, mapEntries, 2)
	mapHostnames := []string{
		mapEntries[0].inputs["hostname"].StringValue(),
		mapEntries[1].inputs["hostname"].StringValue(),
	}
	sort.Strings(mapHostnames)
	require.Equal(t, []string{"*.example.com", "example.com"}, mapHostnames)
	for _, entry := range mapEntries {
		require.Equal(t, "test-cert-map", entry.inputs["map"].StringValue())
		require.Len(t, propertyStrings(entry.inputs["certificates"]), 1)
	}
}

func TestByodHostnameRequestsNormalizesAndSortsCrossServiceOverlap(t *testing.T) {
	got := byodHostnameRequests([]LBServiceEntry{
		{Name: "web", Config: byodService("Example.COM.", "*.example.com", "example.com")},
		{Name: "api", Config: byodService("*.EXAMPLE.COM.", "example.com")},
	})
	require.Equal(t, []byodHostnameRequest{
		{hostname: "*.example.com", services: []string{"api", "web"}},
		{hostname: "example.com", services: []string{"api", "web"}},
	}, got)
}

func TestCreateByodCertsUsesLoadBalancerAuthorizationAndSkipsUnhostedWildcard(t *testing.T) {
	mocks := &byodMocks{}
	entries := []LBServiceEntry{{Name: "api", Config: byodService("API.Example.COM.", "*.example.com")}}
	require.NoError(t, runCreateByodCerts(mocks, entries, nil, true))
	resources := mocks.Resources()

	require.Empty(t, resourcesOfType(resources, dnsAuthorizationType))
	require.Empty(t, resourcesOfType(resources, dnsRecordSetType))
	certificates := resourcesOfType(resources, certificateType)
	require.Len(t, certificates, 1)
	managed := certificates[0].inputs["managed"].ObjectValue()
	require.Equal(t, []string{"api.example.com"}, propertyStrings(managed["domains"]))
	require.False(t, managed.HasValue("dnsAuthorizations"))
	mapEntries := resourcesOfType(resources, certificateMapEntryType)
	require.Len(t, mapEntries, 1)
	require.Equal(t, "api.example.com", mapEntries[0].inputs["hostname"].StringValue())
}

func TestCreateByodCertsDNSAuthorizationWithoutLoadBalancerSkipsOnlyARecord(t *testing.T) {
	mocks := &byodMocks{}
	require.NoError(t, runCreateByodCerts(mocks, []LBServiceEntry{
		{Name: "api", Config: byodService("api.example.com")},
	}, map[string]string{"api.example.com": "trusted-zone"}, false))
	resources := mocks.Resources()

	require.Len(t, resourcesOfType(resources, dnsAuthorizationType), 1)
	require.Len(t, resourcesOfType(resources, certificateType), 1)
	require.Len(t, resourcesOfType(resources, certificateMapEntryType), 1)
	records := resourcesOfType(resources, dnsRecordSetType)
	require.Len(t, records, 1)
	require.Equal(t, byodResourceName("authz", "api.example.com")+"-record", records[0].name)
}

func TestCreateByodCertsRejectsConflictingZonesBeforeRegistration(t *testing.T) {
	mocks := &byodMocks{}
	err := runCreateByodCerts(mocks, []LBServiceEntry{
		{Name: "api", Config: byodService("example.com", "*.example.com")},
	}, map[string]string{
		"example.com":   "zone-z",
		"*.example.com": "zone-a",
	}, true)
	require.ErrorContains(t, err, `BYOD TLS authorization domain "example.com" resolved to "zone-a" and "zone-z"`)
	resources := mocks.Resources()
	require.Empty(t, resourcesOfType(resources, dnsAuthorizationType))
	require.Empty(t, resourcesOfType(resources, dnsRecordSetType))
	require.Empty(t, resourcesOfType(resources, certificateType))
	require.Empty(t, resourcesOfType(resources, certificateMapEntryType))
}

func TestCreateByodCertsReturnsResourceRegistrationFailures(t *testing.T) {
	hostname := "api.example.com"
	certName := byodResourceName("cert", hostname)
	authzName := byodResourceName("authz", hostname)
	for _, tt := range []struct {
		name             string
		failType         string
		failNameContains string
	}{
		{name: "A record", failType: dnsRecordSetType, failNameContains: certName + "-record"},
		{name: "DNS authorization", failType: dnsAuthorizationType, failNameContains: authzName},
		{name: "challenge CNAME", failType: dnsRecordSetType, failNameContains: authzName + "-record"},
		{name: "certificate", failType: certificateType, failNameContains: certName},
		{name: "certificate map entry", failType: certificateMapEntryType, failNameContains: certName + "-map-entry"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mocks := &byodMocks{failType: tt.failType, failNameContains: tt.failNameContains}
			err := runCreateByodCerts(mocks, []LBServiceEntry{
				{Name: "api", Config: byodService(hostname)},
			}, map[string]string{hostname: "trusted-zone"}, true)
			require.ErrorContains(t, err, "injected resource-registration failure")
		})
	}
}

func TestAuthorizationDomainNormalizesWildcardAndExactNames(t *testing.T) {
	for input, want := range map[string]string{
		"Example.COM.":    "example.com",
		"*.Example.COM.":  "example.com",
		"api.example.com": "api.example.com",
	} {
		require.Equal(t, want, authorizationDomain(input))
	}
}
