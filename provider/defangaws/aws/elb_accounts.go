// Ported from https://github.com/DefangLabs/defang-mvp/blob/main/pulumi/shared/aws/elb_accounts.ts
package aws

import (
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// AWS ELB Account IDs.
// See https://docs.aws.amazon.com/elasticloadbalancing/latest/application/enable-access-logging.html
//

var elbAccountIds = map[aws.Region]string{
	"us-east-1":      "127311923021", // US East (N. Virginia)
	"us-east-2":      "033677994240", // US East (Ohio)
	"us-west-1":      "027434742980", // US West (N. California)
	"us-west-2":      "797873946194", // US West (Oregon)
	"af-south-1":     "098369216593", // Africa (Cape Town)
	"ca-central-1":   "985666609251", // Canada (Central)
	"eu-central-1":   "054676820928", // Europe (Frankfurt)
	"eu-west-1":      "156460612806", // Europe (Ireland)
	"eu-west-2":      "652711504416", // Europe (London)
	"eu-south-1":     "635631232127", // Europe (Milan)
	"eu-west-3":      "009996457667", // Europe (Paris)
	"eu-north-1":     "897822967062", // Europe (Stockholm)
	"ap-east-1":      "754344448648", // Asia Pacific (Hong Kong)
	"ap-northeast-1": "582318560864", // Asia Pacific (Tokyo)
	"ap-northeast-2": "600734575887", // Asia Pacific (Seoul)
	"ap-northeast-3": "383597477331", // Asia Pacific (Osaka)
	"ap-southeast-1": "114774131450", // Asia Pacific (Singapore)
	"ap-southeast-2": "783225319266", // Asia Pacific (Sydney)
	"ap-southeast-3": "589379963580", // Asia Pacific (Jakarta)
	"ap-south-1":     "718504428378", // Asia Pacific (Mumbai)
	"me-south-1":     "076674570225", // Middle East (Bahrain)
	"sa-east-1":      "507241528517", // South America (São Paulo)
	"us-gov-west-1":  "048591011584", // AWS GovCloud (US-West); requires a separate account
	"us-gov-east-1":  "190560391635", // AWS GovCloud (US-East); requires a separate account
	"cn-north-1":     "638102146993", // China (Beijing); requires a separate account
	"cn-northwest-1": "037604701340", // China (Ningxia); requires a separate account
}

func getElbAccountId(region aws.Region) string {
	return elbAccountIds[region]
}

func getElbAccountArn(region aws.Region) string {
	accountId := getElbAccountId(region)
	if accountId == "" {
		return ""
	}
	return `arn:aws:iam::` + accountId + `:root`
}

func getElbPrincipal(region aws.Region) any {
	arn := getElbAccountArn(region)
	if arn == "" {
		return getNlbPrincipal()
	}
	return iam.AWSPrincipal{
		AWS: pulumi.String(arn),
	}
}

func getNlbPrincipal() iam.ServicePrincipal {
	return iam.ServicePrincipal{
		Service: pulumi.String("delivery.logs.amazonaws.com"),
	}
}
