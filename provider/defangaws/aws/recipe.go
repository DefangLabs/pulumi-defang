package aws

import "github.com/DefangLabs/pulumi-defang/provider/common"

// AWS recipe config accessors. Cloud-specific keys read from defang-aws:<key>.
var recipe = common.NewRecipe("defang-aws")

var (
	// Alarms enables provider-created CloudWatch alarms on managed databases
	// (Redis/MemoryDB memory+CPU, RDS CPU+storage). Off in the affordable
	// default: alarms have a per-alarm cost and dev stacks don't need paging.
	// Notification wiring is per project, not per recipe: the SNS topic is the
	// AWSConfig.AlarmTopicArn input.
	Alarms                = recipe.Bool("alarms", false)
	AllowBurstable        = recipe.Bool("allow-burstable", true)
	AllowOverwriteRecords = recipe.Bool("allow-overwrite-records", false)
	BackupRetentionDays   = recipe.Int("backup-retention-days", 0)
	BackupWindow          = recipe.String("backup-window", "04:00-05:00")
	BucketKeyEnabled      = recipe.Bool("bucket-key-enabled", true) // minimize KMS costs in non-prod environments
	// ConfigPath is the SSM path prefix ("/…/") for ${VAR} config resolution;
	// empty means the default "/Defang/<project>/<stack>/". Lets deployments
	// keep consuming parameters at a pre-existing path.
	ConfigPath                = recipe.String("config-path", "")
	CreateApexRecord          = recipe.Bool("create-apex-record", true)
	DeletionProtection        = recipe.Bool("deletion-protection", false)
	DeregistrationDelay       = recipe.Int("deregistration-delay", 0)
	FargateCapacityProvider   = recipe.String("fargate-capacity-provider", "FARGATE_SPOT")
	ForceDestroyBucket        = recipe.Bool("force-destroy-bucket", true)
	ForceDestroyHostedzone    = recipe.Bool("force-destroy-hostedzone", false)
	HealthCheckInterval       = recipe.Int("health-check-interval", 5)
	HealthCheckThreshold      = recipe.Int("health-check-threshold", 2)
	HttpRedirectToHttps       = recipe.String("http-redirect-to-https", "HTTP_302")
	LogRetentionDays          = recipe.Int("log-retention-days", 1)
	MinHealthyPercent         = recipe.Int("min-healthy-percent", 0)
	NatGatewayStrategy        = recipe.String("nat-gateway-strategy", "None") // None, Single, or OnePerAz
	NumberOfAvailabilityZones = recipe.Int("number-of-availability-zones", 2)
	RDSNodeType               = recipe.String("rds-node-type", "burstable")
	// RedisEngine selects the managed Redis implementation: "elasticache" or "memorydb".
	RedisEngine          = recipe.String("redis-engine", "elasticache")
	RetainBucketOnDelete = recipe.Bool("retain-bucket-on-delete", false)
	Route53SidecarLogs   = recipe.Bool("route53-sidecar-logs", false)
	RetainDnsOnDelete    = recipe.Bool("retain-dns-on-delete", false)
)
