package aws

import "github.com/DefangLabs/pulumi-defang/provider/common"

// AWS recipe config accessors. Each reads from defang:<key> in stack config.
var (
	AlbAccessLogs           = common.Bool("alb-access-logs", true)
	AllowBurstable          = common.Bool("allow-burstable", true)
	AllowOverwriteRecords   = common.Bool("allow-overwrite-records", false)
	BackupRetentionDays     = common.Int("backup-retention-days", 0)
	BackupWindow            = common.String("backup-window", "04:00-05:00")
	BucketKeyEnabled        = common.Bool("bucket-key-enabled", true)
	CreateApexRecord        = common.Bool("create-apex-record", true)
	DeletionProtection      = common.DeletionProtection
	DeregistrationDelay     = common.Int("deregistration-delay", 0)
	FargateCapacityProvider = common.String("fargate-capacity-provider", "FARGATE_SPOT")
	ForceDestroyHostedzone  = common.Bool("force-destroy-hostedzone", false)
	HealthCheckInterval     = common.Int("health-check-interval", 5)
	HealthCheckThreshold    = common.Int("health-check-threshold", 2)
	HttpRedirectToHttps     = common.String("http-redirect-to-https", "HTTP_302")
	LogRetentionDays        = common.Int("log-retention-days", 1)
	MinHealthyPercent       = common.Int("min-healthy-percent", 0)
	NatGatewayStrategy      = common.String("nat-gateway-strategy", "None") // None, Single, or OnePerAz
	RDSNodeType             = common.String("rds-node-type", "burstable")
	RetainBucketOnDelete    = common.Bool("retain-bucket-on-delete", false)
	Route53SidecarLogs      = common.Bool("route53-sidecar-logs", false)
	RetainDnsOnDelete       = common.Bool("retain-dns-on-delete", false)
	StorageEncrypted        = common.Bool("storage-encrypted", false)
)
