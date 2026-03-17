package aws

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/rds"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type rdsResult struct {
	instance *rds.Instance
}

// nodeInfo describes an RDS instance type's resources and cost.
type nodeInfo struct {
	vcpu int
	gib  float64
	cost float64
}

const costEpsilon = 0.001 // bias for old-generation instances (prefer new gen)

// DB instance type catalogs.
// CPU/Memory/Pricing from:
// https://aws.amazon.com/rds/instance-types/
// https://aws.amazon.com/rds/postgresql/pricing/
// https://instances.vantage.sh

// Burstable instances (t4g Graviton, t3 Intel).
var burstableNodeTypes = map[string]nodeInfo{
	"db.t4g.micro":   {vcpu: 2, gib: 1, cost: 0.016},
	"db.t4g.small":   {vcpu: 2, gib: 2, cost: 0.032},
	"db.t4g.medium":  {vcpu: 2, gib: 4, cost: 0.065},
	"db.t4g.large":   {vcpu: 2, gib: 8, cost: 0.129},
	"db.t4g.xlarge":  {vcpu: 4, gib: 16, cost: 0.258},
	"db.t4g.2xlarge": {vcpu: 8, gib: 32, cost: 0.517},

	"db.t3.micro":   {vcpu: 2, gib: 1, cost: 0.018},
	"db.t3.small":   {vcpu: 2, gib: 2, cost: 0.036},
	"db.t3.medium":  {vcpu: 2, gib: 4, cost: 0.072},
	"db.t3.large":   {vcpu: 2, gib: 8, cost: 0.145},
	"db.t3.xlarge":  {vcpu: 4, gib: 16, cost: 0.29},
	"db.t3.2xlarge": {vcpu: 8, gib: 32, cost: 0.579},
}

// General purpose instances (m7g Graviton, m6g Graviton, m6i Intel).
var generalNodeTypes = map[string]nodeInfo{
	"db.m7g.large":    {vcpu: 2, gib: 8, cost: 0.168},
	"db.m7g.xlarge":   {vcpu: 4, gib: 16, cost: 0.337},
	"db.m7g.2xlarge":  {vcpu: 8, gib: 32, cost: 0.674},
	"db.m7g.4xlarge":  {vcpu: 16, gib: 64, cost: 1.348},
	"db.m7g.8xlarge":  {vcpu: 32, gib: 128, cost: 2.696},
	"db.m7g.12xlarge": {vcpu: 48, gib: 192, cost: 4.044},
	"db.m7g.16xlarge": {vcpu: 64, gib: 256, cost: 5.392},

	"db.m6g.large":    {vcpu: 2, gib: 8, cost: 0.159},
	"db.m6g.xlarge":   {vcpu: 4, gib: 16, cost: 0.318},
	"db.m6g.2xlarge":  {vcpu: 8, gib: 32, cost: 0.636},
	"db.m6g.4xlarge":  {vcpu: 16, gib: 64, cost: 1.272},
	"db.m6g.8xlarge":  {vcpu: 32, gib: 128, cost: 2.544},
	"db.m6g.12xlarge": {vcpu: 48, gib: 192, cost: 3.816},
	"db.m6g.16xlarge": {vcpu: 64, gib: 256, cost: 5.088},

	"db.m6i.large":    {vcpu: 2, gib: 8, cost: 0.178},
	"db.m6i.xlarge":   {vcpu: 4, gib: 16, cost: 0.356},
	"db.m6i.2xlarge":  {vcpu: 8, gib: 32, cost: 0.712},
	"db.m6i.4xlarge":  {vcpu: 16, gib: 64, cost: 1.424},
	"db.m6i.8xlarge":  {vcpu: 32, gib: 128, cost: 2.848},
	"db.m6i.12xlarge": {vcpu: 48, gib: 192, cost: 4.272},
	"db.m6i.16xlarge": {vcpu: 64, gib: 256, cost: 5.696},
	"db.m6i.24xlarge": {vcpu: 96, gib: 384, cost: 8.544},
	"db.m6i.32xlarge": {vcpu: 128, gib: 512, cost: 11.392},
}

// Memory-optimized instances (r7g Graviton, r6g Graviton, r6i Intel).
var memoryOptimizedNodeTypes = map[string]nodeInfo{
	"db.r7g.large":    {vcpu: 2, gib: 16, cost: 0.239},
	"db.r7g.xlarge":   {vcpu: 4, gib: 32, cost: 0.478},
	"db.r7g.2xlarge":  {vcpu: 8, gib: 64, cost: 0.956},
	"db.r7g.4xlarge":  {vcpu: 16, gib: 128, cost: 1.913},
	"db.r7g.8xlarge":  {vcpu: 32, gib: 256, cost: 3.825},
	"db.r7g.12xlarge": {vcpu: 48, gib: 384, cost: 5.738},
	"db.r7g.16xlarge": {vcpu: 64, gib: 512, cost: 7.65},

	"db.r6g.large":    {vcpu: 2, gib: 16, cost: 0.225},
	"db.r6g.xlarge":   {vcpu: 4, gib: 32, cost: 0.45},
	"db.r6g.2xlarge":  {vcpu: 8, gib: 64, cost: 0.899},
	"db.r6g.4xlarge":  {vcpu: 16, gib: 128, cost: 1.798},
	"db.r6g.8xlarge":  {vcpu: 32, gib: 256, cost: 3.597},
	"db.r6g.12xlarge": {vcpu: 48, gib: 384, cost: 5.395},
	"db.r6g.16xlarge": {vcpu: 64, gib: 512, cost: 7.194},

	"db.r6i.large":    {vcpu: 2, gib: 16, cost: 0.25},
	"db.r6i.xlarge":   {vcpu: 4, gib: 32, cost: 0.5},
	"db.r6i.2xlarge":  {vcpu: 8, gib: 64, cost: 1},
	"db.r6i.4xlarge":  {vcpu: 16, gib: 128, cost: 2},
	"db.r6i.8xlarge":  {vcpu: 32, gib: 256, cost: 4},
	"db.r6i.12xlarge": {vcpu: 48, gib: 384, cost: 6},
	"db.r6i.16xlarge": {vcpu: 64, gib: 512, cost: 8},
	"db.r6i.24xlarge": {vcpu: 96, gib: 768, cost: 12},
	"db.r6i.32xlarge": {vcpu: 128, gib: 1024, cost: 16},
}

// nodeTypeCatalogs maps recipe rds-node-type values to their catalog search order.
var nodeTypeCatalogs = map[string][]map[string]nodeInfo{
	"burstable":        {burstableNodeTypes, generalNodeTypes, memoryOptimizedNodeTypes},
	"general":          {generalNodeTypes, memoryOptimizedNodeTypes},
	"memory-optimized": {memoryOptimizedNodeTypes},
}

// rdsInstanceClass returns the cheapest RDS instance class that meets CPU/memory requirements.
// nodeType controls which tiers to consider: "burstable", "general", or "memory-optimized".
func rdsInstanceClass(minCPUs float64, minMemoryMiB int, nodeType string) string {
	minGiB := float64(minMemoryMiB) / 1024

	catalogs, ok := nodeTypeCatalogs[nodeType]
	if !ok {
		catalogs = nodeTypeCatalogs["burstable"]
	}

	for _, catalog := range catalogs {
		if name := cheapestMatch(catalog, minCPUs, minGiB); name != "" {
			return name
		}
	}

	// Fallback: cheapest burstable
	return "db.t4g.micro"
}

// cheapestMatch finds the cheapest instance in catalog with at least minCPUs vCPUs and minGiB memory.
func cheapestMatch(catalog map[string]nodeInfo, minCPUs float64, minGiB float64) string {
	type entry struct {
		name string
		cost float64
	}
	var matches []entry
	minVcpu := int(math.Ceil(minCPUs))
	for name, info := range catalog {
		if info.vcpu >= minVcpu && info.gib >= minGiB {
			matches = append(matches, entry{name, info.cost})
		}
	}
	if len(matches) == 0 {
		return ""
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].cost < matches[j].cost
	})
	return matches[0].name
}

// postgresEngineVersion maps a major version to an RDS engine version string.
func postgresEngineVersion(version int) string {
	switch version {
	case 14:
		return "14"
	case 15:
		return "15"
	case 16:
		return "16"
	case 17:
		return "17"
	default:
		return "17"
	}
}

// createRDS creates a managed RDS Postgres instance for a service.
func createRDS(
	ctx *pulumi.Context,
	serviceName string,
	svc shared.ServiceInput,
	vpcID pulumi.StringOutput,
	privateSubnetIDs pulumi.StringArrayOutput,
	serviceSG *ec2.SecurityGroup,
	recipe Recipe,
	opts ...pulumi.ResourceOption,
) (*rdsResult, error) {
	pg := svc.ResolvePostgres()
	if pg == nil {
		return nil, fmt.Errorf("postgres config is nil")
	}

	// Create DB subnet group
	subnetGroup, err := rds.NewSubnetGroup(ctx, serviceName, &rds.SubnetGroupArgs{
		Name:        pulumi.String(strings.ToLower(serviceName) + "-subnet-group"),
		Description: pulumi.String(fmt.Sprintf("Subnet group for %s postgres", serviceName)),
		SubnetIds:   privateSubnetIDs,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating DB subnet group: %w", err)
	}

	// Create security group for RDS
	rdsSG, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
		VpcId:       vpcID,
		Description: pulumi.String(fmt.Sprintf("RDS security group for %s", serviceName)),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Protocol:       pulumi.String("tcp"),
				FromPort:       pulumi.Int(5432),
				ToPort:         pulumi.Int(5432),
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
		return nil, fmt.Errorf("creating RDS security group: %w", err)
	}

	instanceClass := rdsInstanceClass(svc.GetCPUs(), svc.GetMemoryMiB(), recipe.RDSNodeType)

	// Resolve version: 0 means latest
	engineVersion := postgresEngineVersion(pg.Version)

	rdsArgs := &rds.InstanceArgs{
		AllocatedStorage:         pulumi.Int(20),
		MaxAllocatedStorage:      pulumi.Int(500),
		Engine:                   pulumi.String("postgres"),
		EngineVersion:            pulumi.String(engineVersion),
		InstanceClass:            pulumi.String(instanceClass),
		DbName:                   pulumi.String(pg.DBName),
		Username:                 pulumi.String(pg.Username),
		Password:                 pulumi.String(pg.Password),
		AllowMajorVersionUpgrade: pulumi.Bool(pg.AllowDowntime),
		ApplyImmediately:         pulumi.Bool(pg.AllowDowntime),
		DbSubnetGroupName:        subnetGroup.Name,
		VpcSecurityGroupIds:      pulumi.StringArray{rdsSG.ID()},
		SkipFinalSnapshot:        pulumi.Bool(true),
		PubliclyAccessible:       pulumi.Bool(false),
		DeletionProtection:       pulumi.Bool(recipe.DeletionProtection),
		StorageEncrypted:         pulumi.Bool(recipe.StorageEncrypted),
		AutoMinorVersionUpgrade:  pulumi.Bool(true),
		BackupRetentionPeriod:    pulumi.Int(recipe.BackupRetentionDays),
	}
	if pg.FromSnapshot != "" {
		rdsArgs.SnapshotIdentifier = pulumi.String(pg.FromSnapshot)
	}

	instance, err := rds.NewInstance(ctx, serviceName, rdsArgs, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating RDS instance: %w", err)
	}

	return &rdsResult{
		instance: instance,
	}, nil
}
