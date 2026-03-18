package aws

import (
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/blang/semver"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	awselasticache "github.com/pulumi/pulumi-aws/sdk/v7/go/aws/elasticache"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

type elasticacheResult struct {
	address pulumix.Output[string] // primary or configuration endpoint address
}

// ElastiCache node type catalogs.
// CPU/Memory/Pricing from:
// https://aws.amazon.com/elasticache/pricing/
// https://instances.vantage.sh/?filter=cache

// Burstable cache instances (t4g Graviton, t3 Intel).
var cacheBurstableNodeTypes = map[string]nodeInfo{
	"cache.t4g.micro":  {vcpu: 2, gib: 0.5, cost: 0.016},
	"cache.t4g.small":  {vcpu: 2, gib: 1.37, cost: 0.034},
	"cache.t4g.medium": {vcpu: 2, gib: 3.09, cost: 0.067},

	"cache.t3.micro":  {vcpu: 2, gib: 0.5, cost: 0.017},
	"cache.t3.small":  {vcpu: 2, gib: 1.37, cost: 0.038},
	"cache.t3.medium": {vcpu: 2, gib: 3.09, cost: 0.073},
}

// General-purpose cache instances (m7g Graviton, m6g Graviton).
var cacheGeneralNodeTypes = map[string]nodeInfo{
	"cache.m7g.large":    {vcpu: 2, gib: 13.07, cost: 0.176},
	"cache.m7g.xlarge":   {vcpu: 4, gib: 26.62, cost: 0.352},
	"cache.m7g.2xlarge":  {vcpu: 8, gib: 53.69, cost: 0.704},
	"cache.m7g.4xlarge":  {vcpu: 16, gib: 107.37, cost: 1.408},
	"cache.m7g.8xlarge":  {vcpu: 32, gib: 214.74, cost: 2.816},
	"cache.m7g.12xlarge": {vcpu: 48, gib: 322.11, cost: 4.224},
	"cache.m7g.16xlarge": {vcpu: 64, gib: 429.49, cost: 5.632},

	"cache.m6g.large":    {vcpu: 2, gib: 13.07, cost: 0.166},
	"cache.m6g.xlarge":   {vcpu: 4, gib: 26.62, cost: 0.333},
	"cache.m6g.2xlarge":  {vcpu: 8, gib: 53.69, cost: 0.665},
	"cache.m6g.4xlarge":  {vcpu: 16, gib: 107.37, cost: 1.330},
	"cache.m6g.8xlarge":  {vcpu: 32, gib: 214.74, cost: 2.660},
	"cache.m6g.12xlarge": {vcpu: 48, gib: 322.11, cost: 3.990},
	"cache.m6g.16xlarge": {vcpu: 64, gib: 429.49, cost: 5.320},
}

// Memory-optimized cache instances (r7g Graviton, r6g Graviton).
var cacheMemoryOptimizedNodeTypes = map[string]nodeInfo{
	"cache.r7g.large":    {vcpu: 2, gib: 13.07, cost: 0.239},
	"cache.r7g.xlarge":   {vcpu: 4, gib: 26.62, cost: 0.478},
	"cache.r7g.2xlarge":  {vcpu: 8, gib: 53.69, cost: 0.956},
	"cache.r7g.4xlarge":  {vcpu: 16, gib: 107.37, cost: 1.913},
	"cache.r7g.8xlarge":  {vcpu: 32, gib: 214.74, cost: 3.825},
	"cache.r7g.12xlarge": {vcpu: 48, gib: 322.11, cost: 5.738},
	"cache.r7g.16xlarge": {vcpu: 64, gib: 429.49, cost: 7.650},

	"cache.r6g.large":    {vcpu: 2, gib: 13.07, cost: 0.225},
	"cache.r6g.xlarge":   {vcpu: 4, gib: 26.62, cost: 0.450},
	"cache.r6g.2xlarge":  {vcpu: 8, gib: 53.69, cost: 0.900},
	"cache.r6g.4xlarge":  {vcpu: 16, gib: 107.37, cost: 1.800},
	"cache.r6g.8xlarge":  {vcpu: 32, gib: 214.74, cost: 3.600},
	"cache.r6g.12xlarge": {vcpu: 48, gib: 322.11, cost: 5.400},
	"cache.r6g.16xlarge": {vcpu: 64, gib: 429.49, cost: 7.200},
}

// cacheNodeTypeCatalogs maps recipe node-type values to their catalog search order.
var cacheNodeTypeCatalogs = map[string][]map[string]nodeInfo{
	"burstable":        {cacheBurstableNodeTypes, cacheGeneralNodeTypes, cacheMemoryOptimizedNodeTypes},
	"general":          {cacheGeneralNodeTypes, cacheMemoryOptimizedNodeTypes},
	"memory-optimized": {cacheMemoryOptimizedNodeTypes},
}

// cacheNodeType returns the cheapest ElastiCache node type meeting CPU/memory requirements.
// nodeType controls which tiers to consider: "burstable", "general", or "memory-optimized".
func cacheNodeType(minCPUs float64, minMemoryMiB int, nodeType string) string {
	minGiB := float64(minMemoryMiB) / 1024

	catalogs, ok := cacheNodeTypeCatalogs[nodeType]
	if !ok {
		catalogs = cacheNodeTypeCatalogs["burstable"]
	}

	for _, catalog := range catalogs {
		if name := cheapestMatch(catalog, minCPUs, minGiB); name != "" {
			return name
		}
	}

	// Fallback: cheapest burstable
	return "cache.t4g.micro"
}

// transitEncryptionSupported returns true for valkey or redis >= 7.0.5.
// "Transit encryption mode is not supported for engine version 6.2.6.
// Please use engine version 7.0.5 or higher."
func transitEncryptionSupported(engine, engineVersion string) bool {
	if engine == "valkey" {
		return true
	}
	if engineVersion == "" {
		return false
	}
	v, err := semver.ParseTolerant(engineVersion)
	if err != nil {
		return false
	}
	min705, _ := semver.ParseTolerant("7.0.5")
	return v.GTE(min705)
}

// createElasticache creates a managed ElastiCache Redis/Valkey replication group for a service.
func createElasticache(
	ctx *pulumi.Context,
	_ shared.ConfigProvider,
	serviceName string,
	svc shared.ServiceInput,
	vpcID pulumix.Output[string],
	privateSubnetIDs pulumix.Output[[]string],
	serviceSG *ec2.SecurityGroup,
	recipe Recipe,
	opts ...pulumi.ResourceOption,
) (*elasticacheResult, error) {
	if svc.Redis == nil {
		return nil, fmt.Errorf("redis config is nil")
	}

	// Detect engine (redis vs valkey) from image name.
	image := svc.GetImage()
	engine := "redis"
	if strings.Contains(strings.ToLower(image), "valkey") {
		engine = "valkey"
	}

	// Parse version from image tag (e.g. "7.2" from "redis:7.2").
	engineVersion := ""
	if svc.Image != nil {
		engineVersion = shared.ParseImageTag(*svc.Image)
	}

	allowDowntime := false
	if svc.Redis.AllowDowntime != nil {
		allowDowntime = *svc.Redis.AllowDowntime
	}

	// Redis port: use first declared port or default 6379.
	port := 6379
	if len(svc.Ports) > 0 {
		port = svc.Ports[0].Target
	}

	// Create ElastiCache subnet group (always private).
	subnetGroup, err := awselasticache.NewSubnetGroup(ctx, serviceName, &awselasticache.SubnetGroupArgs{
		Description: pulumi.Sprintf("Subnet group for %s redis", serviceName),
		SubnetIds:   pulumi.StringArrayOutput(privateSubnetIDs),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ElastiCache subnet group: %w", err)
	}

	// Create security group allowing ingress only from the service SG.
	cacheSG, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
		VpcId:       pulumi.StringOutput(vpcID),
		Description: pulumi.String(fmt.Sprintf("ElastiCache security group for %s", serviceName)),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:       pulumi.String("tcp"),
				FromPort:       pulumi.Int(port),
				ToPort:         pulumi.Int(port),
				SecurityGroups: pulumi.StringArray{serviceSG.ID()},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ElastiCache security group: %w", err)
	}

	nodeType := cacheNodeType(svc.GetCPUs(), svc.GetMemoryMiB(), recipe.RDSNodeType)
	replicas := svc.GetReplicas()
	transitEncryption := transitEncryptionSupported(engine, engineVersion)

	rgArgs := &awselasticache.ReplicationGroupArgs{
		ApplyImmediately:         pulumi.Bool(allowDowntime),
		AtRestEncryptionEnabled:  pulumi.Bool(true),
		AutomaticFailoverEnabled: pulumi.Bool(replicas > 1),
		AutoMinorVersionUpgrade:  pulumi.Bool(true),
		Description:              pulumi.String(fmt.Sprintf("Managed %s for %s", engine, serviceName)),
		Engine:                   pulumi.String(engine),
		MultiAzEnabled:           pulumi.Bool(replicas > 1),
		NodeType:                 pulumi.String(nodeType),
		NumCacheClusters:         pulumi.Int(replicas),
		Port:                     pulumi.Int(port),
		SecurityGroupIds:         pulumi.StringArray{cacheSG.ID()},
		SubnetGroupName:          subnetGroup.Name,
		TransitEncryptionEnabled: pulumi.Bool(transitEncryption),
	}
	if engineVersion != "" {
		rgArgs.EngineVersion = pulumi.String(engineVersion)
	}
	if transitEncryption {
		rgArgs.TransitEncryptionMode = pulumi.String("preferred")
	}
	if svc.Redis.FromSnapshot != nil && *svc.Redis.FromSnapshot != "" {
		rgArgs.SnapshotName = pulumi.String(*svc.Redis.FromSnapshot)
	}
	if recipe.BackupRetentionDays > 0 {
		rgArgs.SnapshotRetentionLimit = pulumi.Int(recipe.BackupRetentionDays)
	}

	clusterOpts := append(opts, pulumi.IgnoreChanges([]string{
		"atRestEncryptionEnabled",
		"authToken",
		"authTokenUpdateStrategy",
		"clusterMode",
		"finalSnapshotIdentifier",
		"maintenanceWindow",
		"snapshotWindow",
		"transitEncryptionMode",
	}))
	rg, err := awselasticache.NewReplicationGroup(ctx, serviceName, rgArgs, clusterOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating ElastiCache replication group: %w", err)
	}

	// Use configuration endpoint (cluster mode enabled) if available, else primary endpoint.
	address := pulumix.Apply2(rg.ConfigurationEndpointAddress, rg.PrimaryEndpointAddress,
		func(cfg, primary string) string {
			if cfg != "" {
				return cfg
			}
			return primary
		},
	)

	return &elasticacheResult{address: address}, nil
}
