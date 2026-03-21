package aws

import (
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/blang/semver"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	awselasticache "github.com/pulumi/pulumi-aws/sdk/v7/go/aws/elasticache"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

type ElasticacheResult struct {
	Address pulumix.Output[string] // primary or configuration endpoint address
	Record  *route53.Record        // CNAME record in private hosted zone
}

// ElastiCache node type catalogs.
// CPU/Memory/Pricing from:
// https://aws.amazon.com/elasticache/pricing/
// https://instances.vantage.sh/?filter=cache

// Burstable cache instances (t4g Graviton, t3 Intel).
var cacheBurstableNodeTypes = map[string]nodeInfo{
	"cache.t4g.micro":  {vcpu: 2, gib: 0.5, cost: 0.016},
	"cache.t4g.small":  {vcpu: 2, gib: 1.37, cost: 0.032},
	"cache.t4g.medium": {vcpu: 2, gib: 3.09, cost: 0.065},

	"cache.t3.micro":  {vcpu: 2, gib: 0.5, cost: 0.017},
	"cache.t3.small":  {vcpu: 2, gib: 1.37, cost: 0.034},
	"cache.t3.medium": {vcpu: 2, gib: 3.09, cost: 0.068},
}

// General-purpose small cache instances (m7g.large, m6g.large, m5.large, m4.large).
var cacheGeneralSmallNodeTypes = map[string]nodeInfo{
	"cache.m7g.large": {vcpu: 2, gib: 6.38, cost: 0.158},
	"cache.m6g.large": {vcpu: 2, gib: 6.38, cost: 0.149},
	"cache.m5.large":  {vcpu: 2, gib: 6.38, cost: 0.156},
	"cache.m4.large":  {vcpu: 2, gib: 6.42, cost: 0.156 + costEpsilon},
}

// General-purpose cache instances (m7g, m6g, m5, m4 Graviton/Intel).
var cacheGeneralNodeTypes = map[string]nodeInfo{
	// includes small
	"cache.m7g.large":    {vcpu: 2, gib: 6.38, cost: 0.158},
	"cache.m7g.xlarge":   {vcpu: 4, gib: 12.93, cost: 0.315},
	"cache.m7g.2xlarge":  {vcpu: 8, gib: 26.04, cost: 0.629},
	"cache.m7g.4xlarge":  {vcpu: 16, gib: 52.26, cost: 1.257},
	"cache.m7g.8xlarge":  {vcpu: 32, gib: 103.68, cost: 2.514},
	"cache.m7g.12xlarge": {vcpu: 48, gib: 157.12, cost: 3.77},
	"cache.m7g.16xlarge": {vcpu: 64, gib: 209.55, cost: 5.028},

	"cache.m6g.large":    {vcpu: 2, gib: 6.38, cost: 0.149},
	"cache.m6g.xlarge":   {vcpu: 4, gib: 12.93, cost: 0.297},
	"cache.m6g.2xlarge":  {vcpu: 8, gib: 26.04, cost: 0.593},
	"cache.m6g.4xlarge":  {vcpu: 16, gib: 52.26, cost: 1.186},
	"cache.m6g.8xlarge":  {vcpu: 32, gib: 103.68, cost: 2.372},
	"cache.m6g.12xlarge": {vcpu: 48, gib: 157.12, cost: 3.557},
	"cache.m6g.16xlarge": {vcpu: 64, gib: 209.55, cost: 4.743},

	"cache.m5.large":    {vcpu: 2, gib: 6.38, cost: 0.156},
	"cache.m5.xlarge":   {vcpu: 4, gib: 12.93, cost: 0.311},
	"cache.m5.2xlarge":  {vcpu: 8, gib: 26.04, cost: 0.623},
	"cache.m5.4xlarge":  {vcpu: 16, gib: 52.26, cost: 1.245},
	"cache.m5.12xlarge": {vcpu: 48, gib: 157.12, cost: 3.744},
	"cache.m5.24xlarge": {vcpu: 96, gib: 314.32, cost: 7.488},

	"cache.m4.large":    {vcpu: 2, gib: 6.42, cost: 0.156 + costEpsilon},
	"cache.m4.xlarge":   {vcpu: 4, gib: 14.28, cost: 0.311 + costEpsilon},
	"cache.m4.2xlarge":  {vcpu: 8, gib: 29.7, cost: 0.623 + costEpsilon},
	"cache.m4.4xlarge":  {vcpu: 16, gib: 60.78, cost: 1.245 + costEpsilon},
	"cache.m4.10xlarge": {vcpu: 40, gib: 154.64, cost: 3.112 + costEpsilon},
}

// Memory-optimized cache instances (r7g, r6g, r5, r4).
var cacheMemoryOptimizedNodeTypes = map[string]nodeInfo{
	"cache.r7g.large":    {vcpu: 2, gib: 13.07, cost: 0.219},
	"cache.r7g.xlarge":   {vcpu: 4, gib: 26.32, cost: 0.437},
	"cache.r7g.2xlarge":  {vcpu: 8, gib: 52.82, cost: 0.873},
	"cache.r7g.4xlarge":  {vcpu: 16, gib: 105.81, cost: 1.745},
	"cache.r7g.8xlarge":  {vcpu: 32, gib: 209.55, cost: 3.491},
	"cache.r7g.12xlarge": {vcpu: 48, gib: 317.77, cost: 5.235},
	"cache.r7g.16xlarge": {vcpu: 64, gib: 419.09, cost: 6.981},

	"cache.r6g.large":    {vcpu: 2, gib: 13.07, cost: 0.206},
	"cache.r6g.xlarge":   {vcpu: 4, gib: 26.32, cost: 0.411},
	"cache.r6g.2xlarge":  {vcpu: 8, gib: 52.82, cost: 0.821},
	"cache.r6g.4xlarge":  {vcpu: 16, gib: 105.81, cost: 1.642},
	"cache.r6g.8xlarge":  {vcpu: 32, gib: 209.55, cost: 3.284},
	"cache.r6g.12xlarge": {vcpu: 48, gib: 317.77, cost: 4.925},
	"cache.r6g.16xlarge": {vcpu: 64, gib: 419.09, cost: 6.567},

	"cache.r5.large":    {vcpu: 2, gib: 13.07, cost: 0.216},
	"cache.r5.xlarge":   {vcpu: 4, gib: 26.32, cost: 0.431},
	"cache.r5.2xlarge":  {vcpu: 8, gib: 52.82, cost: 0.862},
	"cache.r5.4xlarge":  {vcpu: 16, gib: 105.81, cost: 1.724},
	"cache.r5.12xlarge": {vcpu: 48, gib: 317.77, cost: 5.184},
	"cache.r5.24xlarge": {vcpu: 96, gib: 635.61, cost: 10.368},

	"cache.r4.large":    {vcpu: 2, gib: 12.3, cost: 0.228},
	"cache.r4.xlarge":   {vcpu: 4, gib: 25.05, cost: 0.455},
	"cache.r4.2xlarge":  {vcpu: 8, gib: 50.47, cost: 0.91},
	"cache.r4.4xlarge":  {vcpu: 16, gib: 101.38, cost: 1.82},
	"cache.r4.8xlarge":  {vcpu: 32, gib: 203.26, cost: 3.64},
	"cache.r4.16xlarge": {vcpu: 64, gib: 407, cost: 7.28},
}

// Memory-optimized cache instances with data tiering (r6gd).
var cacheMemoryOptimizedDataTieringNodeTypes = map[string]nodeInfo{
	"cache.r6gd.xlarge":   {vcpu: 4, gib: 99.33, cost: 0.781},
	"cache.r6gd.2xlarge":  {vcpu: 8, gib: 199.07, cost: 1.56},
	"cache.r6gd.4xlarge":  {vcpu: 16, gib: 398.14, cost: 3.12},
	"cache.r6gd.8xlarge":  {vcpu: 32, gib: 796.28, cost: 6.24},
	"cache.r6gd.12xlarge": {vcpu: 48, gib: 1194.42, cost: 9.358},
	"cache.r6gd.16xlarge": {vcpu: 64, gib: 1592.56, cost: 12.477},
}

// Network-optimized cache instances (c7gn).
var cacheNetworkOptimizedNodeTypes = map[string]nodeInfo{
	"cache.c7gn.large":    {vcpu: 2, gib: 3.09, cost: 0.255},
	"cache.c7gn.xlarge":   {vcpu: 4, gib: 6.38, cost: 0.509},
	"cache.c7gn.2xlarge":  {vcpu: 8, gib: 12.94, cost: 1.018},
	"cache.c7gn.4xlarge":  {vcpu: 16, gib: 26.05, cost: 2.037},
	"cache.c7gn.8xlarge":  {vcpu: 32, gib: 52.26, cost: 4.073},
	"cache.c7gn.12xlarge": {vcpu: 48, gib: 78.56, cost: 6.11},
	"cache.c7gn.16xlarge": {vcpu: 64, gib: 105.81, cost: 8.147},
}

// cacheNodeTypeCatalogs maps recipe node-type values to their catalog search order.
var cacheNodeTypeCatalogs = map[string][]map[string]nodeInfo{
	"burstable":                          {cacheBurstableNodeTypes, cacheGeneralNodeTypes, cacheMemoryOptimizedNodeTypes},
	"general":                            {cacheGeneralNodeTypes, cacheMemoryOptimizedNodeTypes},
	"general-small":                      {cacheGeneralSmallNodeTypes},
	"memory-optimized":                   {cacheMemoryOptimizedNodeTypes},
	"memory-optimized-with-data-tiering": {cacheMemoryOptimizedDataTieringNodeTypes},
	"network-optimized":                  {cacheNetworkOptimizedNodeTypes},
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

// CreateElasticache creates a managed ElastiCache Redis/Valkey replication group for a service.
func CreateElasticache(
	ctx *pulumi.Context,
	_ shared.ConfigProvider,
	serviceName string,
	svc shared.ServiceInput,
	vpcID pulumi.StringInput,
	privateSubnetIDs pulumi.StringArrayInput,
	serviceSG *ec2.SecurityGroup,
	privateZoneId pulumi.IDInput,
	privateFqdn string,
	recipe Recipe,
	deps []pulumi.Resource,
	opts ...pulumi.ResourceOption,
) (*ElasticacheResult, error) {
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

	tags := pulumi.StringMap{
		"defang:service": pulumi.String(serviceName),
	}

	// Create ElastiCache subnet group (always private).
	subnetGroup, err := awselasticache.NewSubnetGroup(ctx, serviceName, &awselasticache.SubnetGroupArgs{
		Description: pulumi.String(common.DefangComment),
		SubnetIds:   privateSubnetIDs.ToStringArrayOutput(),
		Tags:        tags,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ElastiCache subnet group: %w", err)
	}

	// Create security group allowing ingress only from the service SG.
	cacheSG, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
		VpcId:       vpcID.ToStringOutput(),
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
		Tags: tags,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ElastiCache security group: %w", err)
	}

	nodeType := cacheNodeType(svc.GetCPUs(), svc.GetMemoryMiB(), recipe.RDSNodeType)
	replicas := svc.GetReplicas()
	transitEncryption := transitEncryptionSupported(engine, engineVersion)

	// Final snapshot: create one when deletion protection is on (production)
	var finalSnapshotIdentifier pulumi.StringPtrInput
	if recipe.DeletionProtection {
		finalSnapshotIdentifier = pulumi.String(fmt.Sprintf("%s-%s-%s-final", ctx.Project(), ctx.Stack(), serviceName))
	}

	rgArgs := &awselasticache.ReplicationGroupArgs{
		ApplyImmediately:         pulumi.Bool(allowDowntime),
		AtRestEncryptionEnabled:  pulumi.Bool(true),
		AutomaticFailoverEnabled: pulumi.Bool(replicas > 1),
		AutoMinorVersionUpgrade:  pulumi.Bool(true),
		Description:              pulumi.String(common.DefangComment),
		Engine:                   pulumi.String(engine),
		FinalSnapshotIdentifier:  finalSnapshotIdentifier,
		MultiAzEnabled:           pulumi.Bool(replicas > 1),
		NodeType:                 pulumi.String(nodeType),
		NumCacheClusters:         pulumi.Int(replicas),
		Port:                     pulumi.Int(port),
		SecurityGroupIds:         pulumi.StringArray{cacheSG.ID()},
		SubnetGroupName:          subnetGroup.Name,
		Tags:                     tags,
		TransitEncryptionEnabled: pulumi.Bool(transitEncryption),
	}
	if engineVersion != "" {
		rgArgs.EngineVersion = pulumi.String(engineVersion)
	}
	if replicas > 1 && recipe.DeletionProtection {
		rgArgs.ClusterMode = pulumi.String("compatible")
	}
	if transitEncryption {
		rgArgs.TransitEncryptionMode = pulumi.String("preferred")
	}
	if svc.Redis.FromSnapshot != nil && *svc.Redis.FromSnapshot != "" {
		rgArgs.SnapshotName = pulumi.String(*svc.Redis.FromSnapshot)
	}
	if recipe.BackupRetentionDays > 0 {
		rgArgs.SnapshotRetentionLimit = pulumi.Int(recipe.BackupRetentionDays)
		rgArgs.SnapshotWindow = pulumi.String("09:30-10:30")
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
	if len(deps) > 0 {
		clusterOpts = append(clusterOpts, pulumi.DependsOn(deps))
	}
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

	result := &ElasticacheResult{Address: address}

	// Create CNAME record in private hosted zone
	if privateFqdn != "" {
		record, err := route53.NewRecord(ctx, privateFqdn, &route53.RecordArgs{
			ZoneId:  privateZoneId.ToIDOutput().ToStringOutput(),
			Name:    pulumi.String(privateFqdn),
			Type:    pulumi.String("CNAME"),
			Ttl:     pulumi.Int(300),
			Records: pulumi.StringArray{pulumi.StringOutput(address)},
		}, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating CNAME record for %s: %w", serviceName, err)
		}
		result.Record = record
	}

	return result, nil
}
