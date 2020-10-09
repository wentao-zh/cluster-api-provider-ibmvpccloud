package scope

import (
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/go-logr/logr"
	infrav1 "github.com/multicloudlab/cluster-api-provider-ibmvpccloud/api/v1alpha3"
	"github.com/pkg/errors"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterScopeParams struct {
	IBMVPCClients
	Client        client.Client
	Logger        logr.Logger
	Cluster       *clusterv1.Cluster
	IBMVPCCluster *infrav1.IBMVPCCluster
}

type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	IBMVPCClients
	Cluster       *clusterv1.Cluster
	IBMVPCCluster *infrav1.IBMVPCCluster
}

func NewClusterScope(params ClusterScopeParams, iamEndpoint string, apiKey string, svcEndpoint string) (*ClusterScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("failed to generate new scope from nil Cluster")
	}
	if params.IBMVPCCluster == nil {
		return nil, errors.New("failed to generate new scope from nil IBMVPCCluster")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.IBMVPCCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	vpcErr := params.IBMVPCClients.setIBMVPCService(iamEndpoint, svcEndpoint, apiKey)
	if vpcErr != nil {
		return nil, errors.Wrap(vpcErr, "failed to create IBM VPC session")
	}

	return &ClusterScope{
		Logger:        params.Logger,
		client:        params.Client,
		IBMVPCClients: params.IBMVPCClients,
		Cluster:       params.Cluster,
		IBMVPCCluster: params.IBMVPCCluster,
		patchHelper:   helper,
	}, nil
}

func (s *ClusterScope) CreateVPC() (*vpcv1.VPC, error) {
	vpcReply, err := s.ensureVPCUnique(s.IBMVPCCluster.Spec.VPC)
	if err != nil {
		return nil, err
	} else {
		if vpcReply != nil {
			//TODO need a resonable wraped error
			return vpcReply, nil
		}
	}

	options := &vpcv1.CreateVPCOptions{}
	options.SetResourceGroup(&vpcv1.ResourceGroupIdentity{
		ID: &s.IBMVPCCluster.Spec.ResourceGroup,
	})
	options.SetName(s.IBMVPCCluster.Spec.VPC)
	vpc, _, err := s.IBMVPCClients.VPCService.CreateVPC(options)
	if err != nil {
		return nil, err
	} else {
		return vpc, nil
	}
}

func (s *ClusterScope) DeleteVPC() error {
	deleteVpcOptions := &vpcv1.DeleteVPCOptions{}
	deleteVpcOptions.SetID(s.IBMVPCCluster.Status.VPC.ID)
	_, err := s.IBMVPCClients.VPCService.DeleteVPC(deleteVpcOptions)

	return err
}

func (s *ClusterScope) ensureVPCUnique(vpcName string) (*vpcv1.VPC, error) {
	listVpcsOptions := &vpcv1.ListVpcsOptions{}
	vpcs, _, err := s.IBMVPCClients.VPCService.ListVpcs(listVpcsOptions)
	if err != nil {
		return nil, err
	} else {
		for _, vpc := range vpcs.Vpcs {
			if *vpc.Name == vpcName {
				return &vpc, nil
			}
		}
		return nil, nil
	}
}
