package tests

import (
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// awsURN builds a URN for an AWS provider component type.
func awsURN(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type("defang-aws:index:"+typ), "name")
}

// azureURN builds a URN for an Azure provider component type.
func azureURN(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type("defang-azure:defangazure:"+typ), "name")
}

// gcpURN builds a URN for a GCP provider component type.
func gcpURN(typ string) resource.URN {
	return resource.NewURN("stack", "proj", "", tokens.Type("defang-gcp:defanggcp:"+typ), "name")
}

// servicesMap wraps a service map as the top-level Inputs for a Project Construct call.
func servicesMap(services map[string]property.Value) property.Map {
	return property.NewMap(map[string]property.Value{
		"services": property.New(property.NewMap(services)),
	})
}

// serviceWithImage builds a minimal service property value with just an image.
func serviceWithImage(image string) property.Value {
	return property.New(property.NewMap(map[string]property.Value{
		"image": property.New(image),
	}))
}

// serviceWithPorts builds a service property value with an image and one or more port configs.
func serviceWithPorts(image string, ports ...property.Value) property.Value {
	return property.New(property.NewMap(map[string]property.Value{
		"image": property.New(image),
		"ports": property.New(property.NewArray(ports)),
	}))
}

// ingressPort builds a port config property value for an ingress port.
func ingressPort(port int) property.Value {
	return property.New(property.NewMap(map[string]property.Value{
		"target": property.New(float64(port)),
		"mode":   property.New("ingress"),
	}))
}
