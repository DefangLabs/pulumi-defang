package defangscaleway

import (
	"encoding/json"
	"strings"

	"github.com/DefangLabs/defang/src/pkg/deployment"
	providerscaleway "github.com/DefangLabs/pulumi-defang/provider/defangscaleway/scaleway"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const cloudName = "scaleway"

func publicRedisEndpoint(connection pulumi.StringOutput) pulumi.StringOutput {
	endpoint := connection.ApplyT(func(value string) string {
		host := providerscaleway.RedisAddressFromConnectionString(value)
		// The operational contract only needs the host and port. Strip URI
		// query/fragment data as well as the credentials removed above.
		if i := strings.IndexAny(host, "?#"); i >= 0 {
			host = host[:i]
		}
		return host
	}).(pulumi.StringOutput)
	// The upstream SDK marks connectionString secret. Only this sanitized
	// address is public; retaining secrecy would encrypt the whole summary.
	return pulumi.Unsecret(endpoint).(pulumi.StringOutput)
}

func serviceDeployment(
	ctx *pulumi.Context, logicalName string, kind deployment.ServiceKind, etag string,
	id pulumi.IDOutput, name, endpoint pulumi.StringOutput, healthcheckPath string,
) pulumi.StringOutput {
	return pulumi.All(id.ToStringOutput(), name, endpoint).ApplyT(func(values []any) (string, error) {
		service := deployment.Service{
			Kind: kind, ID: values[0].(string), Name: values[1].(string), Endpoint: values[2].(string),
			HealthcheckPath: healthcheckPath,
		}
		if kind == deployment.KindContainer {
			// Pulumi stores regional resource IDs as region/UUID; Cockpit uses the UUID.
			parts := strings.Split(service.ID, "/")
			service.LogLabels = map[string]string{
				"resource_type": "serverless_container", "resource_id": parts[len(parts)-1],
			}
		}
		return (deployment.Metadata{
			Version: deployment.Version, Provider: cloudName, Project: ctx.Project(), Stack: ctx.Stack(),
			Etag: etag, Services: map[string]deployment.Service{logicalName: service},
		}).Encode()
	}).(pulumi.StringOutput)
}

func projectDeployment(ctx *pulumi.Context, etag string, services []any) pulumi.StringOutput {
	return pulumi.All(services...).ApplyT(func(values []any) (string, error) {
		metadata := deployment.Metadata{
			Version: deployment.Version, Provider: cloudName, Project: ctx.Project(), Stack: ctx.Stack(),
			Etag: etag, Summary: true, Services: map[string]deployment.Service{},
		}
		for _, value := range values {
			var service deployment.Metadata
			if err := json.Unmarshal([]byte(value.(string)), &service); err != nil {
				return "", err
			}
			for name, resource := range service.Services {
				metadata.Services[name] = resource
			}
		}
		return metadata.Encode()
	}).(pulumi.StringOutput)
}
