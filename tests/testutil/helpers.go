package testutil

import (
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// AwsURN builds a URN for an AWS provider component type.
func AwsURN(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type("defang-aws:index:"+typ), "name")
}

// AzureURN builds a URN for an Azure provider component type.
func AzureURN(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type("defang-azure:index:"+typ), "name")
}

// GcpURN builds a URN for a GCP provider component type.
func GcpURN(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type("defang-gcp:index:"+typ), "name")
}

// ServicesMap wraps a service map as the top-level Inputs for a Project Construct call.
func ServicesMap(services map[string]property.Value) property.Map {
	return property.NewMap(map[string]property.Value{
		"services": property.New(property.NewMap(services)),
	})
}

// ServiceWithImage builds a minimal service property value with just an image.
func ServiceWithImage(image string) property.Value {
	return property.New(property.NewMap(map[string]property.Value{
		"image": property.New(image),
	}))
}

// ServiceWithPorts builds a service property value with an image and one or more port configs.
func ServiceWithPorts(image string, ports ...property.Value) property.Value {
	return property.New(property.NewMap(map[string]property.Value{
		"image": property.New(image),
		"ports": property.New(property.NewArray(ports)),
	}))
}

// IngressPort builds a port config property value for an ingress port.
func IngressPort(port int) property.Value {
	return property.New(property.NewMap(map[string]property.Value{
		"target": property.New(float64(port)),
		"mode":   property.New("ingress"),
	}))
}

// HostPort builds a port config property value for a host-mode port.
func HostPort(port int) property.Value {
	return property.New(property.NewMap(map[string]property.Value{
		"target": property.New(float64(port)),
		"mode":   property.New("host"),
	}))
}
