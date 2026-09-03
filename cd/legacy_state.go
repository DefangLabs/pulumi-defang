package main

// Safe in-place migration from a legacy Defang deploy driver.
//
// The old and current drivers open the same Pulumi stack. The current program
// therefore has to alias every data-bearing legacy resource it keeps; without
// an alias Pulumi creates an empty replacement and deletes the original. This
// preflight reads only resource identity from the old state, maps supported
// databases to the matching compose services, and passes their exact URNs to
// the current providers as Pulumi aliases.
//
// Aliases are the migration mechanism. The guard is defense in depth: `up`
// stops when a database cannot be mapped one-to-one, the state cannot be read,
// another cloud owns it, or the current image cannot operate on a legacy
// resource. `preview` receives every alias that can be proved safe but remains
// non-blocking so an operator can inspect the complete plan. `down` and
// `destroy` remain deliberately unguarded because their stated purpose is to
// delete the selected stack.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/DefangLabs/pulumi-defang/cd/program"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"go.yaml.in/yaml/v4"
)

// allowTakeoverConfigKey is an operator-only break-glass override. Its value
// is the exact "<project>/<stack>" target, not a boolean: Fabric recipes are
// tenant/mode scoped, so a boolean would disarm every stack sharing a recipe.
const allowTakeoverConfigKey = "defang:allowLegacyStateTakeover"

// allowTakeoverEnv provides the same exact-stack override for a CD started by
// hand. The CLI does not forward it during ordinary deployments.
const allowTakeoverEnv = "DEFANG_ALLOW_LEGACY_STATE_TAKEOVER"

const migrationRunbook = "https://github.com/DefangLabs/pulumi-defang/blob/main/docs/legacy-cd-migration.md"

const maxReportedResources = 5

const (
	cloudAWS   = "aws"
	cloudAzure = "azure"
	cloudGCP   = "gcp"
)

// thisCDTopLevelNames is the historical union of names registered outside the
// Project component. Never remove an entry: an older current-CD state may
// still contain it. A match also requires the expected cloud package and root
// position, so switching a stack between AWS/GCP/Azure cannot pass as a normal
// current-CD update.
var thisCDTopLevelNames = map[string]bool{
	"project-pb":            true,
	"self-destruct":         true,
	"defang-self-destruct":  true,
	"self-destruct-starter": true,
}

var thisCDPackages = map[string]string{
	cloudAWS:   "defang-aws",
	cloudAzure: "defang-azure",
	cloudGCP:   "defang-gcp",
}

var nativePackages = map[string]string{
	cloudAWS:   "aws",
	cloudAzure: "azure-native",
	cloudGCP:   "gcp",
}

// These legacy resources cannot be loaded by the current minimal CD images.
// Allowing `up` would create the new graph and then fail in the delete phase,
// leaving two partial deployments. Keep them blocked until their deletion or
// adoption path is implemented.
var unsupportedLegacyTypes = map[string]string{
	"cloudbuild:index:CloudBuild":    "the GCP image does not contain the legacy private cloudbuild plugin",
	"pulumi-nodejs:dynamic:Resource": "the AWS image does not contain the legacy Node dynamic provider/runtime",
}

// replaceableLegacyTypes is an allowlist of stateless or reconstructible
// resources emitted by the released legacy drivers. Unknown types fail closed:
// a package match alone is not proof that deletion is safe. Stateful resources
// are deliberately absent and need an exact alias rule above instead.
var replaceableLegacyTypes = map[string]bool{
	// Legacy AWS networking, compute, build, identity, routing, and certificates.
	"aws:acm/certificate:Certificate":                             true,
	"aws:acm/certificateValidation:CertificateValidation":         true,
	"aws:cloudwatch/eventRule:EventRule":                          true,
	"aws:cloudwatch/eventTarget:EventTarget":                      true,
	"aws:cloudwatch/logGroup:LogGroup":                            true,
	"aws:cloudwatch/logResourcePolicy:LogResourcePolicy":          true,
	"aws:codebuild/project:Project":                               true,
	"aws:ec2/defaultSecurityGroup:DefaultSecurityGroup":           true,
	"aws:ec2/eip:Eip":                                             true,
	"aws:ec2/internetGateway:InternetGateway":                     true,
	"aws:ec2/natGateway:NatGateway":                               true,
	"aws:ec2/route:Route":                                         true,
	"aws:ec2/routeTable:RouteTable":                               true,
	"aws:ec2/routeTableAssociation:RouteTableAssociation":         true,
	"aws:ec2/securityGroup:SecurityGroup":                         true,
	"aws:ec2/subnet:Subnet":                                       true,
	"aws:ec2/vpc:Vpc":                                             true,
	"aws:ec2/vpcDhcpOptions:VpcDhcpOptions":                       true,
	"aws:ec2/vpcDhcpOptionsAssociation:VpcDhcpOptionsAssociation": true,
	"aws:ec2/vpcEndpoint:VpcEndpoint":                             true,
	"aws:ecr/lifecyclePolicy:LifecyclePolicy":                     true,
	"aws:ecr/pullThroughCacheRule:PullThroughCacheRule":           true,
	"aws:ecr/repository:Repository":                               true,
	"aws:ecs/cluster:Cluster":                                     true,
	"aws:ecs/clusterCapacityProviders:ClusterCapacityProviders":   true,
	"aws:ecs/service:Service":                                     true,
	"aws:ecs/taskDefinition:TaskDefinition":                       true,
	"aws:iam/instanceProfile:InstanceProfile":                     true,
	"aws:iam/policy:Policy":                                       true,
	"aws:iam/role:Role":                                           true,
	"aws:iam/rolePoliciesExclusive:RolePoliciesExclusive":         true,
	"aws:iam/rolePolicy:RolePolicy":                               true,
	"aws:iam/rolePolicyAttachment:RolePolicyAttachment":           true,
	"aws:lambda/function:Function":                                true,
	"aws:lambda/permission:Permission":                            true,
	"aws:lb/listener:Listener":                                    true,
	"aws:lb/listenerCertificate:ListenerCertificate":              true,
	"aws:lb/listenerRule:ListenerRule":                            true,
	"aws:lb/loadBalancer:LoadBalancer":                            true,
	"aws:lb/targetGroup:TargetGroup":                              true,
	"aws:lb/targetGroupAttachment:TargetGroupAttachment":          true,
	"aws:resourcegroups/group:Group":                              true,
	"aws:route53/record:Record":                                   true,
	"aws:route53/zone:Zone":                                       true,
	"aws:s3/bucketPolicy:BucketPolicy":                            true,
	"aws:s3/bucketPublicAccessBlock:BucketPublicAccessBlock":      true,
	"aws:scheduler/schedule:Schedule":                             true,
	"aws:wafv2/webAcl:WebAcl":                                     true,
	"aws:wafv2/webAclAssociation:WebAclAssociation":               true,
	"awsx:ec2:Vpc":                                                true,
	"awsx:ecr:Repository":                                         true,
	"defang-mvp:shared/ecs/defang:Defang":                         true,
	"tls:index/privateKey:PrivateKey":                             true,
	"tls:index/selfSignedCert:SelfSignedCert":                     true,

	// Legacy GCP networking, compute, build, identity, routing, and certificates.
	"gcp:artifactregistry/repository:Repository":                        true,
	"gcp:artifactregistry/repositoryIamBinding:RepositoryIamBinding":    true,
	"gcp:certificatemanager/certificate:Certificate":                    true,
	"gcp:certificatemanager/certificateMap:CertificateMap":              true,
	"gcp:certificatemanager/certificateMapEntry:CertificateMapEntry":    true,
	"gcp:certificatemanager/dnsAuthorization:DnsAuthorization":          true,
	"gcp:cloudrunv2/service:Service":                                    true,
	"gcp:cloudrunv2/serviceIamMember:ServiceIamMember":                  true,
	"gcp:compute/address:Address":                                       true,
	"gcp:compute/backendService:BackendService":                         true,
	"gcp:compute/firewall:Firewall":                                     true,
	"gcp:compute/forwardingRule:ForwardingRule":                         true,
	"gcp:compute/globalAddress:GlobalAddress":                           true,
	"gcp:compute/globalForwardingRule:GlobalForwardingRule":             true,
	"gcp:compute/healthCheck:HealthCheck":                               true,
	"gcp:compute/instanceTemplate:InstanceTemplate":                     true,
	"gcp:compute/network:Network":                                       true,
	"gcp:compute/regionBackendService:RegionBackendService":             true,
	"gcp:compute/regionInstanceGroupManager:RegionInstanceGroupManager": true,
	"gcp:compute/regionNetworkEndpointGroup:RegionNetworkEndpointGroup": true,
	"gcp:compute/regionTargetHttpProxy:RegionTargetHttpProxy":           true,
	"gcp:compute/regionUrlMap:RegionUrlMap":                             true,
	"gcp:compute/router:Router":                                         true,
	"gcp:compute/routerNat:RouterNat":                                   true,
	"gcp:compute/subnetwork:Subnetwork":                                 true,
	"gcp:compute/targetHttpProxy:TargetHttpProxy":                       true,
	"gcp:compute/targetHttpsProxy:TargetHttpsProxy":                     true,
	"gcp:compute/uRLMap:URLMap":                                         true,
	"gcp:dns/managedZone:ManagedZone":                                   true,
	"gcp:dns/recordSet:RecordSet":                                       true,
	"gcp:projects/iAMMember:IAMMember":                                  true,
	"gcp:projects/service:Service":                                      true,
	"gcp:secretmanager/secretIamMember:SecretIamMember":                 true,
	"gcp:serviceaccount/account:Account":                                true,
	"gcp:servicenetworking/connection:Connection":                       true,
	"gcp:storage/bucketIAMMember:BucketIAMMember":                       true,
}

var (
	errInvalidMigrationCompose = errors.New("invalid compose project for legacy-state migration")
	errInvalidDeployment       = errors.New("unreadable Pulumi deployment")
)

type serviceKind string

const (
	servicePostgres serviceKind = "Postgres"
	serviceRedis    serviceKind = "Redis"
)

// legacyAliasSpec connects one old custom-resource type to the child the
// current provider registers. legacySuffixes describe names used by released
// legacy drivers; currentSuffix describes the current child's logical name.
type legacyAliasSpec struct {
	cloud          string
	resourceType   string
	serviceKind    serviceKind
	aliasKind      string
	legacySuffixes []string
	currentSuffix  string
	dataBearing    bool
}

var legacyAliasSpecs = []legacyAliasSpec{
	// Legacy TypeScript AWS CD. Its stack prefix varied by deployment, so the
	// matcher accepts an exact service name or a name ending in the service.
	{cloudAWS, "aws:rds/instance:Instance", servicePostgres, compose.AliasInstance, []string{"-postgres", ""}, "", true},
	{cloudAWS, "aws:rds/subnetGroup:SubnetGroup", servicePostgres, compose.AliasSubnetGroup, []string{""}, "", false},
	{cloudAWS, "aws:ec2/securityGroup:SecurityGroup", servicePostgres,
		compose.AliasSecurityGroup, []string{"-postgres", ""}, "", false},
	{cloudAWS, "aws:elasticache/replicationGroup:ReplicationGroup", serviceRedis,
		compose.AliasCluster, []string{""}, "", true},
	{cloudAWS, "aws:elasticache/subnetGroup:SubnetGroup", serviceRedis, compose.AliasSubnetGroup, []string{""}, "", false},
	{cloudAWS, "aws:ec2/securityGroup:SecurityGroup", serviceRedis,
		compose.AliasSecurityGroup, []string{"-redis", ""}, "", false},
	{cloudAWS, "aws:memorydb/cluster:Cluster", serviceRedis, compose.AliasCluster, []string{""}, "", true},
	{cloudAWS, "aws:memorydb/subnetGroup:SubnetGroup", serviceRedis, compose.AliasSubnetGroup, []string{""}, "", false},
	{cloudAWS, "aws:memorydb/parameterGroup:ParameterGroup", serviceRedis,
		compose.AliasParameterGroup, []string{""}, "", false},

	// Legacy Go GCP CD. Released states retain these project/service names even
	// after that driver itself added aliases for an earlier naming scheme.
	{cloudGCP, "gcp:sql/databaseInstance:DatabaseInstance", servicePostgres,
		compose.AliasInstance, []string{"-postgres"}, "", true},
	{cloudGCP, "gcp:sql/user:User", servicePostgres, compose.AliasUser, []string{"-postgres-user"}, "-user", false},
	{cloudGCP, "gcp:sql/database:Database", servicePostgres,
		compose.AliasDatabase, []string{"-postgres-db"}, "-db", false},
	{cloudGCP, "gcp:redis/instance:Instance", serviceRedis, compose.AliasInstance, []string{"-redis"}, "", true},
}

type resourceIdentity struct {
	urn    resource.URN
	typ    string
	parent resource.URN
}

func identityOf(res apitype.ResourceV3) resourceIdentity {
	return resourceIdentity{urn: res.URN, typ: string(res.Type), parent: res.Parent}
}

func (r resourceIdentity) display() string {
	return fmt.Sprintf("%s::%s", r.urn.QualifiedType(), r.urn.Name())
}

func resourcePackage(typ string) string {
	pkg, _, _ := strings.Cut(typ, ":")
	return pkg
}

func isStructural(res resourceIdentity) bool {
	qualified := res.urn.QualifiedType()
	return qualified == resource.RootStackType || strings.HasPrefix(string(qualified), "pulumi:providers:")
}

func isRootChild(res resourceIdentity) bool {
	if strings.Contains(string(res.urn.QualifiedType()), "$") {
		return false
	}
	return res.parent == "" || res.parent.QualifiedType() == resource.RootStackType
}

func isThisCD(res resourceIdentity, cloud string) bool {
	wantPackage := thisCDPackages[cloud]
	for _, typ := range strings.Split(string(res.urn.QualifiedType()), "$") {
		if resourcePackage(typ) == wantPackage {
			return true
		}
	}
	return isRootChild(res) && thisCDTopLevelNames[res.urn.Name()] &&
		resourcePackage(res.typ) == nativePackages[cloud]
}

func foreignResources(resources []resourceIdentity, cloud string) []resourceIdentity {
	var foreign []resourceIdentity
	for _, res := range resources {
		if isStructural(res) || isThisCD(res, cloud) {
			continue
		}
		foreign = append(foreign, res)
	}
	return foreign
}

type stackExporter interface {
	Export(ctx context.Context) (apitype.UntypedDeployment, error)
}

func takeoverAllowed(recipePulumiConfig, project, stack string) bool {
	if project == "" || stack == "" {
		return false
	}
	target := project + "/" + stack
	return recipeTakeoverTarget(recipePulumiConfig) == target || os.Getenv(allowTakeoverEnv) == target
}

func recipeTakeoverTarget(recipePulumiConfig string) string {
	if recipePulumiConfig == "" {
		return ""
	}
	config := configMap{}
	if err := unmarshalRecipe(recipePulumiConfig, config); err != nil {
		return ""
	}
	target, _ := config[allowTakeoverConfigKey].Value.(string)
	return target
}

type desiredService struct {
	kind        serviceKind
	aliases     map[string]string
	environment compose.Environment
}

func desiredManagedServices(composeYAML []byte) (map[string]desiredService, error) {
	var project compose.Project
	if err := yaml.Unmarshal(composeYAML, &project); err != nil {
		return nil, fmt.Errorf("parsing compose for migration: %w", err)
	}
	services := make(map[string]desiredService)
	for name, svc := range project.Services {
		switch {
		case svc.Postgres != nil && svc.Redis != nil:
			return nil, fmt.Errorf("%w: service %q declares both Postgres and Redis", errInvalidMigrationCompose, name)
		case svc.Postgres != nil:
			services[name] = desiredService{kind: servicePostgres, aliases: svc.Aliases, environment: svc.Environment}
		case svc.Redis != nil:
			services[name] = desiredService{kind: serviceRedis, aliases: svc.Aliases, environment: svc.Environment}
		}
	}
	return services, nil
}

type migrationProblem struct {
	resource resourceIdentity
	reason   string
}

type migrationPlan struct {
	aliases      program.ServiceAliases
	blockers     []migrationProblem
	foreignCount int
	adoptedCount int
	recognized   bool
}

func specsFor(cloud, typ string) []legacyAliasSpec {
	var specs []legacyAliasSpec
	for _, spec := range legacyAliasSpecs {
		if spec.cloud == cloud && spec.resourceType == typ {
			specs = append(specs, spec)
		}
	}
	return specs
}

func resourceCloud(typ string) string {
	switch resourcePackage(typ) {
	case "aws", "awsx", "defang-aws", "defang-mvp", "pulumi-nodejs", "tls":
		return cloudAWS
	case "azure-native", "defang-azure":
		return cloudAzure
	case "cloudbuild", "gcp", "defang-gcp":
		return cloudGCP
	case "digitalocean":
		return "digitalocean"
	default:
		return ""
	}
}

func isReplaceableLegacyResource(res resourceIdentity) bool {
	switch res.typ {
	case "aws:s3/bucket:Bucket":
		// The released AWS driver used this bucket only for expiring ALB logs.
		return res.urn.Name() == "alb-logs"
	case "aws:s3/bucketObject:BucketObject", "gcp:storage/bucketObject:BucketObject":
		// This is Defang's generated project metadata, not customer object data.
		return res.urn.Name() == "state"
	default:
		return replaceableLegacyTypes[res.typ]
	}
}

func legacyNameScore(name, service string, suffixes []string) int {
	best := 0
	for _, suffix := range suffixes {
		candidate := service + suffix
		switch {
		case name == candidate:
			best = max(best, 10_000+len(candidate))
		case strings.HasSuffix(name, "-"+candidate):
			best = max(best, len(candidate))
		}
	}
	return best
}

func matchLegacyService(res resourceIdentity, spec legacyAliasSpec, services map[string]desiredService) (string, int) {
	bestName, bestScore := "", 0
	explicitName, explicitMatches := "", 0
	for name, svc := range services {
		if svc.kind != spec.serviceKind {
			continue
		}
		if svc.aliases[spec.aliasKind] == string(res.urn) {
			explicitName = name
			explicitMatches++
			continue
		}
		score := legacyNameScore(res.urn.Name(), name, spec.legacySuffixes)
		if score > bestScore {
			bestName, bestScore = name, score
		} else if score != 0 && score == bestScore {
			bestName = "" // ambiguous; the guard must not guess
		}
	}
	if explicitMatches == 1 {
		return explicitName, 1_000_000
	}
	if explicitMatches > 1 {
		return "", 1_000_000
	}
	return bestName, bestScore
}

func currentTargetKey(res resourceIdentity, spec legacyAliasSpec, services map[string]desiredService) string {
	for name, svc := range services {
		if svc.kind == spec.serviceKind && res.urn.Name() == name+spec.currentSuffix {
			return spec.resourceType + "\x00" + name + "\x00" + spec.aliasKind
		}
	}
	return ""
}

func aliasTargetKey(spec legacyAliasSpec, service string) string {
	return spec.resourceType + "\x00" + service + "\x00" + spec.aliasKind
}

// aliasTargetEnabled mirrors conditional child registration in the current
// providers. An alias is not protective unless the Pulumi program will
// actually register a resource that consumes it.
func aliasTargetEnabled(spec legacyAliasSpec, service desiredService) bool {
	if spec.cloud != cloudGCP || spec.serviceKind != servicePostgres {
		return true
	}
	switch spec.aliasKind {
	case compose.AliasUser:
		password, _ := compose.StaticEnvValue(service.environment["POSTGRES_PASSWORD"])
		return password != nil && *password != ""
	case compose.AliasDatabase:
		database, _ := compose.StaticEnvValue(service.environment["POSTGRES_DB"])
		return database != nil && *database != "" && *database != compose.DEFAULT_POSTGRES_DB
	default:
		return true
	}
}

func awsRedisEngine(config configMap) string {
	engine, _ := config["defang-aws:redis-engine"].Value.(string)
	if engine == "" {
		return "elasticache"
	}
	return engine
}

func specMatchesAWSRedisEngine(spec legacyAliasSpec, config configMap) bool {
	if spec.cloud != cloudAWS || spec.serviceKind != serviceRedis || !spec.dataBearing {
		return true
	}
	engine := awsRedisEngine(config)
	if strings.Contains(spec.resourceType, "memorydb/") {
		return engine == "memorydb"
	}
	return engine == "elasticache"
}

func resolveLegacyAlias(
	res resourceIdentity,
	services map[string]desiredService,
	cloud string,
	config configMap,
) (legacyAliasSpec, string, string) {
	if reason := unsupportedLegacyTypes[res.typ]; reason != "" {
		return legacyAliasSpec{}, "", reason
	}
	if owner := resourceCloud(res.typ); owner == "" || owner != cloud {
		return legacyAliasSpec{}, "", "the resource is not from the selected " +
			strings.ToUpper(cloud) + " legacy driver"
	}
	if cloud == cloudAzure {
		return legacyAliasSpec{}, "", "no legacy Azure in-place adoption path is defined"
	}

	specs := specsFor(cloud, res.typ)
	if len(specs) == 0 {
		if isReplaceableLegacyResource(res) {
			return legacyAliasSpec{}, "", ""
		}
		return legacyAliasSpec{}, "", "the legacy resource type has no reviewed replacement or adoption rule"
	}

	var chosen legacyAliasSpec
	service, bestScore := "", 0
	dataBearing := false
	for _, spec := range specs {
		dataBearing = dataBearing || spec.dataBearing
		name, score := matchLegacyService(res, spec, services)
		if score > bestScore {
			chosen, service, bestScore = spec, name, score
		} else if score != 0 && score == bestScore {
			service = ""
		}
	}
	if service == "" {
		if dataBearing {
			return legacyAliasSpec{}, "", "no unique matching managed service exists in the requested compose project"
		}
		if isReplaceableLegacyResource(res) {
			return legacyAliasSpec{}, "", ""
		}
		return legacyAliasSpec{}, "", "the legacy resource type has no reviewed replacement or adoption rule"
	}
	if !aliasTargetEnabled(chosen, services[service]) {
		return legacyAliasSpec{}, "",
			"the requested compose project will not register a resource to consume this alias"
	}
	if !specMatchesAWSRedisEngine(chosen, config) {
		return legacyAliasSpec{}, "", "the requested AWS Redis engine is a different resource type"
	}
	return chosen, service, ""
}

func analyzeLegacyState(
	resources []resourceIdentity,
	services map[string]desiredService,
	cloud string,
	config configMap,
) migrationPlan {
	plan := migrationPlan{aliases: program.ServiceAliases{}}
	foreign := foreignResources(resources, cloud)
	plan.foreignCount = len(foreign)
	if len(foreign) == 0 {
		return plan
	}

	currentTargets := map[string]bool{}
	for _, res := range resources {
		if !isThisCD(res, cloud) {
			continue
		}
		for _, spec := range specsFor(cloud, res.typ) {
			if key := currentTargetKey(res, spec, services); key != "" {
				currentTargets[key] = true
			}
		}
	}

	seenTargets := map[string]bool{}
	for _, res := range foreign {
		chosen, service, reason := resolveLegacyAlias(res, services, cloud, config)
		if reason != "" {
			plan.blockers = append(plan.blockers, migrationProblem{res, reason})
			continue
		}
		if service == "" {
			continue
		}

		key := aliasTargetKey(chosen, service)
		if currentTargets[key] || seenTargets[key] {
			plan.blockers = append(plan.blockers, migrationProblem{
				res, "the current state already has a resource for this service and alias target",
			})
			continue
		}
		if configured := services[service].aliases[chosen.aliasKind]; configured != "" && configured != string(res.urn) {
			plan.blockers = append(plan.blockers, migrationProblem{
				res, "x-defang-aliases points this service and kind at a different resource",
			})
			continue
		}

		if plan.aliases[service] == nil {
			plan.aliases[service] = map[string]string{}
		}
		plan.aliases[service][chosen.aliasKind] = string(res.urn)
		seenTargets[key] = true
		plan.adoptedCount++
		plan.recognized = plan.recognized || chosen.dataBearing
	}

	// Do not reinterpret an arbitrary flat cloud stack as a Defang migration.
	// A supported legacy state must contain at least one data-bearing resource
	// that maps to the desired compose project. AWS states may alternatively be
	// identified by the legacy Defang component marker.
	if !plan.recognized {
		for _, res := range foreign {
			if strings.Contains(string(res.urn.QualifiedType()), "defang-mvp:") {
				plan.recognized = true
				break
			}
		}
	}
	if !plan.recognized {
		plan.blockers = append(plan.blockers, migrationProblem{
			foreign[0], "the state cannot be identified as a supported legacy Defang deployment",
		})
	}
	return plan
}

func mergeDetectedAliases(dst program.ServiceAliases, src program.ServiceAliases) {
	for service, aliases := range src {
		if dst[service] == nil {
			dst[service] = map[string]string{}
		}
		for kind, urn := range aliases {
			dst[service][kind] = urn
		}
	}
}

func warnMigrationProblems(header string, blockers []migrationProblem) {
	warn(header)
	for i, blocker := range blockers {
		if i == maxReportedResources {
			warn(fmt.Sprintf("  ...and %d more", len(blockers)-maxReportedResources))
			break
		}
		warn(fmt.Sprintf("  %s — %s", blocker.resource.display(), blocker.reason))
	}
}

// prepareLegacyState populates aliases for both preview and up. enforce is
// true only for up: preview applies no infrastructure changes and must remain
// available to show the exact plan, even when up would stop.
func prepareLegacyState(
	ctx context.Context,
	exporter stackExporter,
	recipePulumiConfig, projectName, stackName string,
	composeYAML []byte,
	cloud string,
	config configMap,
	aliases program.ServiceAliases,
	enforce bool,
) error {
	override := takeoverAllowed(recipePulumiConfig, projectName, stackName)
	if override {
		warn("Warning: the legacy-state takeover override is active for " + projectName + "/" + stackName + ".")
	}

	services, err := desiredManagedServices(composeYAML)
	if err != nil {
		return err
	}
	deployment, err := exporter.Export(ctx)
	if err != nil {
		return handleStateInspectionFailure(override, enforce, "could not read the existing state")
	}
	resources, err := resourceIdentitiesIn(deployment)
	if err != nil {
		return handleStateInspectionFailure(override, enforce, "could not parse the existing state")
	}

	plan := analyzeLegacyState(resources, services, cloud, config)
	mergeDetectedAliases(aliases, plan.aliases)
	if summary := stableAliasSummary(plan.aliases); len(summary) != 0 {
		warn("Prepared Pulumi migration aliases: " + strings.Join(summary, ", "))
	}
	if plan.foreignCount == 0 {
		return nil
	}

	if len(plan.blockers) != 0 {
		switch {
		case !enforce:
			warnMigrationProblems(fmt.Sprintf(
				"Preview found %d legacy migration blocker(s); a real up will stop. See %s",
				len(plan.blockers), migrationRunbook,
			), plan.blockers)
			return nil
		case override:
			warnMigrationProblems(fmt.Sprintf(
				"Warning: continuing despite %d legacy migration blocker(s). Unaliased databases may be deleted.",
				len(plan.blockers),
			), plan.blockers)
			return nil
		default:
			return &legacyStateError{plan: plan}
		}
	}

	warn(fmt.Sprintf(
		"Migrating this legacy stack in place: adopting %d resource(s) by Pulumi alias; "+
			"%d other legacy resource(s) may be replaced.",
		plan.adoptedCount, plan.foreignCount-plan.adoptedCount,
	))
	return nil
}

func handleStateInspectionFailure(override, enforce bool, reason string) error {
	if !enforce {
		warn("Preview could not prepare legacy aliases: " + reason +
			". A real up will stop unless an exact-stack override is active.")
		return nil
	}
	if override {
		warn("Warning: continuing without legacy aliases because " + reason + ". Databases may be deleted.")
		return nil
	}
	return &stateInspectionError{reason: reason}
}

func resourceIdentitiesIn(deployment apitype.UntypedDeployment) ([]resourceIdentity, error) {
	if len(deployment.Deployment) == 0 {
		return nil, nil
	}
	if deployment.Version != apitype.DeploymentSchemaVersionCurrent {
		return nil, fmt.Errorf("%w: unsupported version %d", errInvalidDeployment, deployment.Version)
	}
	// Export includes decrypted inputs/outputs. Keep that snapshot local and
	// immediately project it to URN/type/parent; no secret-bearing value is
	// returned, logged, stored in an error, or passed to the Pulumi program.
	var snapshot apitype.DeploymentV3
	if err := json.Unmarshal(deployment.Deployment, &snapshot); err != nil {
		return nil, err
	}
	identities := make([]resourceIdentity, 0, len(snapshot.Resources))
	for i, res := range snapshot.Resources {
		if !res.URN.IsValid() {
			return nil, fmt.Errorf("%w: resource %d has an invalid URN", errInvalidDeployment, i)
		}
		if res.Parent != "" && !res.Parent.IsValid() {
			return nil, fmt.Errorf("%w: resource %d has an invalid parent URN", errInvalidDeployment, i)
		}
		identities = append(identities, identityOf(res))
	}
	return identities, nil
}

type stateInspectionError struct {
	reason string
}

func (e *stateInspectionError) Error() string {
	return "cannot verify that the existing databases are safe to migrate: " + e.reason +
		". Nothing has been changed. Retry, or contact Defang support. See " + migrationRunbook
}

type legacyStateError struct {
	plan migrationPlan
}

func (e *legacyStateError) Error() string {
	var b strings.Builder
	fmt.Fprintf(
		&b,
		"this stack cannot yet be migrated in place safely: %d blocker(s) remain after preparing %d Pulumi alias(es):\n",
		len(e.plan.blockers), e.plan.adoptedCount,
	)
	for i, blocker := range e.plan.blockers {
		if i == maxReportedResources {
			fmt.Fprintf(&b, "  ...and %d more\n", len(e.plan.blockers)-maxReportedResources)
			break
		}
		fmt.Fprintf(&b, "  %s — %s\n", blocker.resource.display(), blocker.reason)
	}
	b.WriteString("\nContinuing could replace or orphan existing infrastructure, including databases and their data. " +
		"Nothing has been changed.\n" +
		"\nRun `preview` to inspect the migration plan, then follow\n  " + migrationRunbook + "\n" +
		"\nDo not run `down` or `destroy` to clear this error: both intentionally delete the selected stack.\n" +
		"\nIf the runbook does not cover this state, contact Defang support.")
	return b.String()
}

// stableAliasSummary is safe for logs: it includes resource identity only,
// never the exported resource inputs or outputs that can hold secrets.
func stableAliasSummary(aliases program.ServiceAliases) []string {
	var summary []string
	for service, kinds := range aliases {
		for kind, urn := range kinds {
			u := resource.URN(urn)
			summary = append(summary, service+"/"+kind+"="+string(u.QualifiedType())+"::"+u.Name())
		}
	}
	sort.Strings(summary)
	return summary
}
