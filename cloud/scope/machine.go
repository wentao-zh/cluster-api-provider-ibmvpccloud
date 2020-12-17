package scope

import (
	"context"
	"fmt"

	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/go-logr/logr"
	infrav1 "github.com/multicloudlab/cluster-api-provider-ibmvpccloud/api/v1alpha3"
	"github.com/pkg/errors"
	"k8s.io/klog/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MachineScopeParams struct {
	IBMVPCClients
	Client        client.Client
	Logger        logr.Logger
	Cluster       *clusterv1.Cluster
	Machine       *clusterv1.Machine
	IBMVPCCluster *infrav1.IBMVPCCluster
	IBMVPCMachine *infrav1.IBMVPCMachine
}

type MachineScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper

	IBMVPCClients
	Cluster *clusterv1.Cluster
	Machine *clusterv1.Machine

	IBMVPCCluster *infrav1.IBMVPCCluster
	IBMVPCMachine *infrav1.IBMVPCMachine
}

func NewMachineScope(params MachineScopeParams, iamEndpoint string, apiKey string, svcEndpoint string) (*MachineScope, error) {
	if params.Machine == nil {
		return nil, errors.New("failed to generate new scope from nil Machine")
	}
	if params.IBMVPCMachine == nil {
		return nil, errors.New("failed to generate new scope from nil IBMVPCCluster")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	helper, err := patch.NewHelper(params.IBMVPCMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	vpcErr := params.IBMVPCClients.setIBMVPCService(iamEndpoint, svcEndpoint, apiKey)
	if vpcErr != nil {
		return nil, errors.Wrap(vpcErr, "failed to create IBM VPC session")
	}

	return &MachineScope{
		Logger:        params.Logger,
		client:        params.Client,
		IBMVPCClients: params.IBMVPCClients,
		Cluster:       params.Cluster,
		IBMVPCCluster: params.IBMVPCCluster,
		patchHelper:   helper,
		Machine:       params.Machine,
		IBMVPCMachine: params.IBMVPCMachine,
	}, nil
}

func (m *MachineScope) CreateMachine() (*vpcv1.Instance, error) {
	instanceReply, err := m.ensureInstanceUnique(m.IBMVPCMachine.Spec.Name)
	if err != nil {
		return nil, err
	} else {
		if instanceReply != nil {
			//TODO need a resonable wraped error
			fmt.Printf("MachineScope CreateMachine:%s,%s\n", *instanceReply.Name, *instanceReply.ID)
			return instanceReply, nil
		}
	}

	options := &vpcv1.CreateInstanceOptions{}
	options.SetInstancePrototype(&vpcv1.InstancePrototype{
		Name: &m.IBMVPCMachine.Spec.Name,
		Image: &vpcv1.ImageIdentity{
			ID: &m.IBMVPCMachine.Spec.Image,
		},
		Profile: &vpcv1.InstanceProfileIdentity{
			Name: &m.IBMVPCMachine.Spec.Profile,
		},
		Zone: &vpcv1.ZoneIdentity{
			Name: &m.IBMVPCMachine.Spec.Zone,
		},
		PrimaryNetworkInterface: &vpcv1.NetworkInterfacePrototype{
			Subnet: &vpcv1.SubnetIdentity{
				ID: &m.IBMVPCMachine.Spec.PrimaryNetworkInterface.Subnet,
			},
		},
	})
	instance, _, err := m.IBMVPCClients.VPCService.CreateInstance(options)
	return instance, err
}

func (m *MachineScope) ensureInstanceUnique(instanceName string) (*vpcv1.Instance, error) {
	options := &vpcv1.ListInstancesOptions{}
	instances, _, err := m.IBMVPCClients.VPCService.ListInstances(options)

	if err != nil {
		return nil, err
	} else {
		for _, instance := range instances.Instances {
			if *instance.Name == instanceName {
				return &instance, nil
			}
		}
		return nil, nil
	}
}

func (m *MachineScope) GetMachine(instanceID string) (*vpcv1.Instance, error) {
	options := &vpcv1.GetInstanceOptions{}
	options.SetID(instanceID)

	instance, _, err := m.IBMVPCClients.VPCService.GetInstance(options)
	return instance, err
}

// PatchObject persists the cluster configuration and status.
func (m *MachineScope) PatchObject() error {
	return m.patchHelper.Patch(context.TODO(), m.IBMVPCMachine)
}

// Close closes the current scope persisting the cluster configuration and status.
func (m *MachineScope) Close() error {
	return m.PatchObject()
}
