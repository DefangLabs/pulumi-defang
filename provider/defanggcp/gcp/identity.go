package gcp

import (
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ServiceIdentity is the service account a service runs as. Account is non-nil
// only when the provider created the account itself; when the caller supplies
// an existing service account (by email), Account is nil and the caller owns
// all IAM role grants for it.
type ServiceIdentity struct {
	Account *serviceaccount.Account
	Email   pulumi.StringInput
}

// deleteOpts returns options for IAM bindings on this identity. Bindings on a
// provider-created account are additionally tied to the account's lifetime.
func (si *ServiceIdentity) deleteOpts() []pulumi.ResourceOption {
	opts := []pulumi.ResourceOption{
		// membership is not a distinct resource, so we risk deleting the
		// membership we are trying to create
		pulumi.DeleteBeforeReplace(true),
	}
	if si.Account != nil {
		// prevent "service account does not exist" errors on down; the binding
		// is removed automatically when the account is deleted
		opts = append(opts, pulumi.DeletedWith(si.Account))
	}
	return opts
}
