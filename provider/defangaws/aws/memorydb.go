package aws

import (
	"fmt"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/memorydb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// MemoryDB node type catalogs. Costs are approximate on-demand us-west-2
// prices, used only for cheapest-match ordering within a catalog.
// https://aws.amazon.com/memorydb/pricing/

// Burstable MemoryDB instances (t4g Graviton). db.t4g.small is the smallest
// available MemoryDB node type.
var memoryDBBurstableNodeTypes = map[string]nodeInfo{
	"db.t4g.small":  {vcpu: 2, gib: 1.37, cost: 0.048},
	"db.t4g.medium": {vcpu: 2, gib: 3.09, cost: 0.096},
}

// Memory-optimized MemoryDB instances (r7g, r6g Graviton).
var memoryDBMemoryOptimizedNodeTypes = map[string]nodeInfo{
	"db.r7g.large":    {vcpu: 2, gib: 13.07, cost: 0.412},
	"db.r7g.xlarge":   {vcpu: 4, gib: 26.32, cost: 0.824},
	"db.r7g.2xlarge":  {vcpu: 8, gib: 52.82, cost: 1.648},
	"db.r7g.4xlarge":  {vcpu: 16, gib: 105.81, cost: 3.297},
	"db.r7g.8xlarge":  {vcpu: 32, gib: 209.55, cost: 6.594},
	"db.r7g.12xlarge": {vcpu: 48, gib: 317.77, cost: 9.89},
	"db.r7g.16xlarge": {vcpu: 64, gib: 419.09, cost: 13.188},

	"db.r6g.large":    {vcpu: 2, gib: 13.07, cost: 0.395},
	"db.r6g.xlarge":   {vcpu: 4, gib: 26.32, cost: 0.79},
	"db.r6g.2xlarge":  {vcpu: 8, gib: 52.82, cost: 1.58},
	"db.r6g.4xlarge":  {vcpu: 16, gib: 105.81, cost: 3.16},
	"db.r6g.8xlarge":  {vcpu: 32, gib: 209.55, cost: 6.32},
	"db.r6g.12xlarge": {vcpu: 48, gib: 317.77, cost: 9.48},
	"db.r6g.16xlarge": {vcpu: 64, gib: 419.09, cost: 12.64},
}

// memoryDBNodeTypeCatalogs maps recipe node-type values to their catalog
// search order. MemoryDB has no general-purpose (m-family) tier.
var memoryDBNodeTypeCatalogs = map[string][]map[string]nodeInfo{
	"burstable":        {memoryDBBurstableNodeTypes, memoryDBMemoryOptimizedNodeTypes},
	"memory-optimized": {memoryDBMemoryOptimizedNodeTypes},
}

// memoryDBNodeType returns the cheapest MemoryDB node type meeting CPU/memory
// requirements; see cacheNodeType.
func memoryDBNodeType(minCPUs float64, minMemoryMiB int, nodeType string) string {
	minGiB := float64(minMemoryMiB) / 1024

	catalogs, ok := memoryDBNodeTypeCatalogs[nodeType]
	if !ok {
		catalogs = memoryDBNodeTypeCatalogs["burstable"]
	}

	for _, catalog := range catalogs {
		if name := cheapestMatch(catalog, minCPUs, minGiB); name != "" {
			return name
		}
	}

	// Fallback: smallest burstable
	return "db.t4g.small"
}

// memoryDBParameterGroupFamily derives the parameter group family from the
// engine and its major version, e.g. memorydb_redis7 or memorydb_valkey8.
func memoryDBParameterGroupFamily(engine, engineVersion string) string {
	major := "7"
	if engineVersion != "" {
		major = strings.SplitN(engineVersion, ".", 2)[0]
	}
	return fmt.Sprintf("memorydb_%s%s", engine, major)
}

// CreateMemoryDB creates a managed MemoryDB Redis/Valkey cluster for a
// service. Selected over ElastiCache via the redis-engine recipe; the result
// shape matches CreateElasticache so callers are engine-agnostic.
//
//nolint:funlen
func CreateMemoryDB(
	ctx *pulumi.Context,
	serviceName string,
	svc compose.ServiceConfig,
	vpcID pulumi.StringInput,
	privateSubnetIDs pulumi.StringArrayInput,
	privateSgID pulumi.StringPtrInput,
	alarmTopicArn pulumi.StringInput,
	deps []pulumi.Resource,
	opts ...pulumi.ResourceOption,
) (*ElasticacheResult, error) {
	if svc.Redis == nil {
		return nil, ErrRedisConfigNil
	}

	engine, engineVersion := detectRedisEngine(svc)

	// Redis port: use first declared port or default 6379.
	port := 6379
	if len(svc.Ports) > 0 {
		port = int(svc.Ports[0].Target)
	}

	tags := pulumi.StringMap{
		"defang:service": pulumi.String(serviceName),
	}

	// MemoryDB subnet group (always private). Subnet IDs of an in-use subnet
	// group can't be updated in place.
	var subnetGroupName pulumi.StringPtrOutput
	if privateSubnetIDs != nil {
		sgOpts := common.MergeOptions(opts, svc.AliasOptions(compose.AliasSubnetGroup)...)
		subnetGroup, err := memorydb.NewSubnetGroup(ctx, serviceName, &memorydb.SubnetGroupArgs{
			Description: pulumi.String(common.DefangComment),
			SubnetIds:   privateSubnetIDs,
			Tags:        tags,
		}, common.MergeOptions(sgOpts, pulumi.ReplaceOnChanges([]string{"subnetIds"}))...)
		if err != nil {
			return nil, fmt.Errorf("creating MemoryDB subnet group: %w", err)
		}
		subnetGroupName = subnetGroup.Name.ToStringPtrOutput()
	}

	pgOpts := common.MergeOptions(opts, svc.AliasOptions(compose.AliasParameterGroup)...)
	parameterGroup, err := memorydb.NewParameterGroup(ctx, serviceName, &memorydb.ParameterGroupArgs{
		Description: pulumi.String(common.DefangComment),
		Family:      pulumi.String(memoryDBParameterGroupFamily(engine, engineVersion)),
		Parameters: memorydb.ParameterGroupParameterArray{
			&memorydb.ParameterGroupParameterArgs{
				Name:  pulumi.String("maxmemory-policy"),
				Value: pulumi.String("allkeys-lru"),
			},
		},
		Tags: tags,
	}, pgOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating MemoryDB parameter group: %w", err)
	}

	// The service SG is only referenced when supplied; standalone Construct
	// callers (e.g. unit tests) may omit it.
	var ingressSGs pulumi.StringArray
	if privateSgID != nil {
		ingressSGs = pulumi.StringArray{privateSgID.ToStringPtrOutput().Elem()}
	}
	secOpts := common.MergeOptions(opts, svc.AliasOptions(compose.AliasSecurityGroup)...)
	cacheSG, err := ec2.NewSecurityGroup(ctx, serviceName, &ec2.SecurityGroupArgs{
		VpcId:       vpcID.ToStringOutput(),
		Description: pulumi.String("MemoryDB security group for " + serviceName),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Description:    pulumi.String("Allow incoming Redis traffic"),
				Protocol:       pulumi.String("tcp"),
				FromPort:       pulumi.Int(port),
				ToPort:         pulumi.Int(port),
				SecurityGroups: ingressSGs,
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Description: pulumi.String("Allow all outbound traffic"),
				Protocol:    pulumi.String("-1"),
				FromPort:    pulumi.Int(0),
				ToPort:      pulumi.Int(0),
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Tags: tags,
	}, common.MergeOptions(secOpts,
		pulumi.Timeouts(&pulumi.CustomTimeouts{Delete: "2m"}),
	)...)
	if err != nil {
		return nil, fmt.Errorf("creating MemoryDB security group: %w", err)
	}

	nodeType := memoryDBNodeType(svc.GetCPUs(), svc.GetMemoryMiB(), RDSNodeType.Get(ctx))
	replicas := svc.GetReplicas()
	deletionProtection := DeletionProtection.Get(ctx)

	// Final snapshot: create one when deletion protection is on (production)
	var finalSnapshotName pulumi.StringPtrInput
	if deletionProtection {
		finalSnapshotName = pulumi.String(fmt.Sprintf("%s-%s-%s-final", ctx.Project(), ctx.Stack(), serviceName))
	}

	clusterArgs := &memorydb.ClusterArgs{
		// open-access = no AUTH, matching the ElastiCache path's no-authToken
		// posture (see the authToken TODO in elasticache.go); isolation comes
		// from private subnets + the SG below. TLS stays enabled (AWS default).
		AclName:                 pulumi.String("open-access"),
		AutoMinorVersionUpgrade: pulumi.Bool(true),
		Description:             pulumi.String(common.DefangComment),
		Engine:                  pulumi.String(engine),
		FinalSnapshotName:       finalSnapshotName,
		NodeType:                pulumi.String(nodeType),
		NumReplicasPerShard:     pulumi.Int(replicas - 1),
		NumShards:               pulumi.Int(1),
		ParameterGroupName:      parameterGroup.Name,
		Port:                    pulumi.Int(port),
		SecurityGroupIds:        pulumi.StringArray{cacheSG.ID()},
		SubnetGroupName:         subnetGroupName,
		Tags:                    tags,
	}
	if engineVersion != "" {
		clusterArgs.EngineVersion = pulumi.String(engineVersion)
	}
	if svc.Redis.FromSnapshot != nil && *svc.Redis.FromSnapshot != "" {
		clusterArgs.SnapshotName = pulumi.String(*svc.Redis.FromSnapshot)
	}
	if backupRetentionDays := BackupRetentionDays.Get(ctx); backupRetentionDays > 0 {
		clusterArgs.SnapshotRetentionLimit = pulumi.Int(backupRetentionDays)
		clusterArgs.SnapshotWindow = pulumi.String("09:30-10:30")
	}

	clusterOpts := common.MergeOptions(opts, svc.AliasOptions(compose.AliasCluster)...)
	clusterOpts = common.MergeOptions(clusterOpts, pulumi.IgnoreChanges([]string{
		"finalSnapshotName",
		"maintenanceWindow",
		"snapshotWindow",
	}))
	if len(deps) > 0 {
		clusterOpts = append(clusterOpts, pulumi.DependsOn(deps))
	}
	cluster, err := memorydb.NewCluster(ctx, serviceName, clusterArgs, clusterOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating MemoryDB cluster: %w", err)
	}

	err = createDBAlarms(ctx, serviceName, "AWS/MemoryDB",
		pulumi.StringMap{"ClusterName": cluster.Name}, tags, alarmTopicArn, []dbAlarm{
			{
				suffix:             "memory-usage",
				metricName:         "DatabaseCapacityUsagePercentage",
				comparisonOperator: "GreaterThanThreshold",
				threshold:          80,
				statistic:          "Maximum",
				description:        "MemoryDB memory usage has exceeded 80%",
			},
			{
				suffix:             "cpu-usage",
				metricName:         "CPUUtilization",
				comparisonOperator: "GreaterThanThreshold",
				threshold:          80,
				statistic:          "Maximum",
				description:        "MemoryDB CPU usage has exceeded 80%",
			},
		}, opts...)
	if err != nil {
		return nil, err
	}

	address := cluster.ClusterEndpoints.Index(pulumi.Int(0)).Address().Elem()

	return &ElasticacheResult{Address: address, ClusterID: cluster.Name.ToStringOutput()}, nil
}
