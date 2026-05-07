module defang-scaleway

go 1.25.9

replace github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-scaleway => ../../sdk/v2/go/defang-scaleway

require (
	github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-scaleway v0.0.0-00010101000000-000000000000
	github.com/pulumi/pulumi/sdk/v3 v3.231.0
)
