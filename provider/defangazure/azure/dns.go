package azure

import (
	"fmt"

	"github.com/pulumi/pulumi-azure-native-sdk/privatedns/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DNSResult holds private DNS zones for a project.
type DNSResult struct {
	// PostgresZoneName is the private DNS zone used by PostgreSQL Flexible Servers
	// for automatic hostname registration (e.g. "projectname.private.postgres.database.azure.com").
	PostgresZoneName pulumi.StringOutput

	// PostgresZoneID is the resource ID of the postgres private DNS zone,
	// needed when creating a Flexible Server with VNet integration.
	PostgresZoneID pulumi.StringOutput

	// ServiceZone is the "internal" private DNS zone for short-name service records
	// (e.g. CNAME "db.internal" → full postgres FQDN).
	ServiceZone *privatedns.PrivateZone
}

// CreateDNSZones creates two private DNS zones linked to the project VNet:
//  1. A postgres zone ("<name>.private.postgres.database.azure.com") for Flexible Server integration.
//  2. An "internal" zone for human-readable service CNAMEs (e.g. "db.internal").
func CreateDNSZones(
	ctx *pulumi.Context,
	name string,
	infra *SharedInfra,
	networking *NetworkingResult,
	opts ...pulumi.ResourceOption,
) (*DNSResult, error) {
	// Postgres private DNS zone — name must end in ".private.postgres.database.azure.com".
	pgZoneName := name + ".private.postgres.database.azure.com"
	pgZone, err := privatedns.NewPrivateZone(ctx, name+"-pg-dns", &privatedns.PrivateZoneArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.String("global"),
		PrivateZoneName:   pulumi.String(pgZoneName),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating postgres private DNS zone: %w", err)
	}

	_, err = privatedns.NewVirtualNetworkLink(ctx, name+"-pg-dns-link", &privatedns.VirtualNetworkLinkArgs{
		ResourceGroupName:    infra.ResourceGroup.Name,
		PrivateZoneName:      pgZone.Name,
		RegistrationEnabled:  pulumi.Bool(false),
		VirtualNetwork:       &privatedns.SubResourceArgs{Id: networking.VNet.ID().ToStringOutput()},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating postgres DNS VNet link: %w", err)
	}

	// "internal" zone for short service names (e.g. "db.internal").
	svcZone, err := privatedns.NewPrivateZone(ctx, name+"-svc-dns", &privatedns.PrivateZoneArgs{
		ResourceGroupName: infra.ResourceGroup.Name,
		Location:          pulumi.String("global"),
		PrivateZoneName:   pulumi.String("internal"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating internal private DNS zone: %w", err)
	}

	_, err = privatedns.NewVirtualNetworkLink(ctx, name+"-svc-dns-link", &privatedns.VirtualNetworkLinkArgs{
		ResourceGroupName:    infra.ResourceGroup.Name,
		PrivateZoneName:      svcZone.Name,
		RegistrationEnabled:  pulumi.Bool(false),
		VirtualNetwork:       &privatedns.SubResourceArgs{Id: networking.VNet.ID().ToStringOutput()},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating service DNS VNet link: %w", err)
	}

	return &DNSResult{
		PostgresZoneName: pgZone.Name,
		PostgresZoneID:   pgZone.ID().ToStringOutput(),
		ServiceZone:      svcZone,
	}, nil
}

// AddPostgresDNSRecord creates a CNAME record in the "internal" zone pointing
// serviceName (e.g. "db") to the postgres server's fully-qualified domain name.
func AddPostgresDNSRecord(
	ctx *pulumi.Context,
	serviceName string,
	postgresFQDN pulumi.StringOutput,
	dns *DNSResult,
	infra *SharedInfra,
	opts ...pulumi.ResourceOption,
) error {
	_, err := privatedns.NewPrivateRecordSet(ctx, serviceName+"-dns", &privatedns.PrivateRecordSetArgs{
		ResourceGroupName:    infra.ResourceGroup.Name,
		PrivateZoneName:      dns.ServiceZone.Name,
		RecordType:           pulumi.String("CNAME"),
		RelativeRecordSetName: pulumi.String(serviceName),
		Ttl:                  pulumi.Float64(300),
		CnameRecord: &privatedns.CnameRecordArgs{
			Cname: postgresFQDN,
		},
	}, opts...)
	if err != nil {
		return fmt.Errorf("creating DNS CNAME for %s: %w", serviceName, err)
	}
	return nil
}
