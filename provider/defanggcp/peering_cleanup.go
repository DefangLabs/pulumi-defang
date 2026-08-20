package defanggcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/pulumi/pulumi-go-provider/infer"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PeeringCleanup deletes the VPC network peering that a Service Networking
// connection leaves behind.
//
// It exists because Pulumi cannot delete the connection itself. The provider's
// delete calls servicenetworking `deleteConnection`, which fails while the
// producer still holds resources — and producer cleanup lags the instance
// delete, by up to 4 days for Cloud SQL:
// https://docs.cloud.google.com/vpc/docs/configure-private-services-access.
// See also hashicorp/terraform-provider-google#16275, whose "fix" was an abandon
// policy rather than a working delete, and #18834, still open, which records that
// Google's own console removes the peering through the Compute API instead.
//
// So the connection carries DeletionPolicy: ABANDON (see createVPCPeeringInfra)
// and this resource removes the peering with the call the console uses. Doing it
// as a resource rather than in the CD keeps the VPC's name in this resource's own
// inputs — nothing has to be recovered from a state snapshot — and leaves the
// ordering to Pulumi: the managed instances depend on this resource, so they are
// deleted before the peering goes, and the peering goes before the reserved range
// and the VPC that it holds.
//
// GCP does warn against deleting a private connection this way, because the
// connection record survives and a later CreateConnection on the same network
// with different ranges then fails. That does not apply here: this runs only on
// the way to deleting the network itself, so no later connection can be created
// on it.
//
// Create is deliberately a no-op. The connection resource creates the peering;
// this resource only knows how to remove it.
type PeeringCleanup struct{}

type PeeringCleanupArgs struct {
	// ProjectId is the GCP project holding the network.
	ProjectId string `provider:"replaceOnChanges" pulumi:"projectId"`
	// Network is the VPC to remove peerings from, as a name, an id or a
	// self-link — everything up to the last path separator is ignored.
	Network string `provider:"replaceOnChanges" pulumi:"network"`
}

type PeeringCleanupState struct {
	PeeringCleanupArgs
}

func (*PeeringCleanup) Annotate(a infer.Annotator) {
	a.Describe(&PeeringCleanup{}, "Removes the VPC network peering left behind by an abandoned Service "+
		"Networking connection. Creating it does nothing; deleting it calls compute.networks.removePeering, "+
		"which is the only delete GCP honours while the service producer is still releasing its resources.")
}

func (a *PeeringCleanupArgs) Annotate(an infer.Annotator) {
	an.Describe(&a.ProjectId, "The GCP project that holds the network.")
	an.Describe(&a.Network, "The VPC to remove peerings from: a name, an id or a self-link.")
}

func (*PeeringCleanup) Create(
	ctx context.Context,
	req infer.CreateRequest[PeeringCleanupArgs],
) (infer.CreateResponse[PeeringCleanupState], error) {
	return infer.CreateResponse[PeeringCleanupState]{
		ID:     req.Name,
		Output: PeeringCleanupState{PeeringCleanupArgs: req.Inputs},
	}, nil
}

// Delete removes every peering on the network. Nothing else ever peers with a
// Defang project VPC, so each one is the abandoned connection's.
func (*PeeringCleanup) Delete(
	ctx context.Context,
	req infer.DeleteRequest[PeeringCleanupState],
) (infer.DeleteResponse, error) {
	network := networkName(req.State.Network)
	if network == "" {
		return infer.DeleteResponse{}, nil
	}
	return infer.DeleteResponse{}, removeNetworkPeerings(ctx, req.State.ProjectId, network)
}

// networkName takes the VPC's name from a name, an id
// (projects/P/global/networks/NAME) or a self-link.
func networkName(network string) string {
	return network[strings.LastIndex(network, "/")+1:]
}

func removeNetworkPeerings(ctx context.Context, gcpProject, network string) error {
	client, err := compute.NewNetworksRESTClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create compute client: %w", err)
	}
	defer func() { _ = client.Close() }()

	net, err := client.Get(ctx, &computepb.GetNetworkRequest{Project: gcpProject, Network: network})
	if err != nil {
		if isNotFoundErr(err) {
			return nil // the VPC is already gone, and its peerings with it
		}
		return fmt.Errorf("failed to read the VPC %s: %w", network, err)
	}

	for _, peering := range net.GetPeerings() {
		name := peering.GetName()
		op, err := client.RemovePeering(ctx, &computepb.RemovePeeringNetworkRequest{
			Project: gcpProject,
			Network: network,
			NetworksRemovePeeringRequestResource: &computepb.NetworksRemovePeeringRequest{
				Name: &name,
			},
		})
		if err == nil {
			err = op.Wait(ctx)
		}
		if err != nil && !isNotFoundErr(err) {
			return fmt.Errorf("failed to remove the peering %s from the VPC %s: %w", name, network, err)
		}
	}
	return nil
}

// isNotFoundErr reports whether err is a 404 from either transport, in which
// case what it names is already deleted.
func isNotFoundErr(err error) bool {
	var gerr *googleapi.Error
	if errors.As(err, &gerr) && gerr.Code == 404 {
		return true
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
		return true
	}
	return false
}
