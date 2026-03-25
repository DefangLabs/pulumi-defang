package aws

import "github.com/DefangLabs/pulumi-defang/provider/common"

// AWS recipe config accessors. Each reads from defang:<key> in stack config.
var (
	AllowBurstable          = common.Bool("allow-burstable", true)
	AllowOverwriteRecords   = common.Bool("allow-overwrite-records", false)
	BackupRetentionDays     = common.Int("backup-retention-days", 0)
	BackupWindow            = common.String("backup-window", "04:00-05:00")
	DeletionProtection      = common.DeletionProtection
	DeregistrationDelay     = common.Int("deregistration-delay", 0)
	FargateCapacityProvider = common.String("fargate-capacity-provider", "FARGATE_SPOT")
	ForceDestroyHostedzone  = common.Bool("force-destroy-hostedzone", false)
	HealthCheckInterval     = common.Int("health-check-interval", 5)
	HealthCheckThreshold    = common.Int("health-check-threshold", 2)
	LogRetentionDays        = common.Int("log-retention-days", 1)
	MinHealthyPercent       = common.Int("min-healthy-percent", 0)
	NatGatewayStrategy      = common.String("nat-gateway-strategy", "Single")
	RDSNodeType             = common.String("rds-node-type", "burstable")
	RetainDnsOnDelete       = common.Bool("retain-dns-on-delete", false)
	StorageEncrypted        = common.Bool("storage-encrypted", false)
)
