package controllers

import (
	"fmt"

	"github.com/choerodon/choerodon-cluster-agent/pkg/polaris/config"
	"github.com/choerodon/choerodon-cluster-agent/pkg/polaris/kube"
	kubeAPICoreV1 "k8s.io/api/core/v1"
	kubeAPIMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Interface is an interface for k8s controllers (e.g. Deployments and StatefulSets)
type Interface interface {
	GetName() string
	GetNamespace() string
	GetPodTemplate() *kubeAPICoreV1.PodTemplateSpec
	GetPodSpec() *kubeAPICoreV1.PodSpec
	GetKind() config.SupportedController
	GetObjectMeta() kubeAPIMetaV1.ObjectMeta
}

// GenericController is a base implementation with some free methods for inherited structs
type GenericController struct {
	Name      string
	Namespace string
}

// GetName is inherited by all controllers using generic controller to get the name of the controller
func (g GenericController) GetName() string {
	return g.Name
}

// GetNamespace is inherited by all controllers using generic controller to get the namespace of the controller
func (g GenericController) GetNamespace() string {
	return g.Namespace
}

// LoadControllersByKind loads a list of controllers from the kubeResources by detecting their type
func LoadControllersByKind(controllerKind config.SupportedController, kubeResources *kube.ResourceProvider) ([]Interface, error) {
	interfaces := []Interface{}
	switch controllerKind {
	case config.Deployments:
		for _, deploy := range kubeResources.Deployments {
			interfaces = append(interfaces, NewDeploymentController(deploy))
		}
	case config.StatefulSets:
		for _, statefulSet := range kubeResources.StatefulSets {
			interfaces = append(interfaces, NewStatefulSetController(statefulSet))
		}
	case config.DaemonSets:
		for _, daemonSet := range kubeResources.DaemonSets {
			interfaces = append(interfaces, NewDaemonSetController(daemonSet))
		}
	case config.Jobs:
		for _, job := range kubeResources.Jobs {
			interfaces = append(interfaces, NewJobController(job))
		}
	case config.CronJobs:
		for _, cronJob := range kubeResources.CronJobs {
			interfaces = append(interfaces, NewCronJobController(cronJob))
		}
	case config.ReplicationControllers:
		for _, replicationController := range kubeResources.ReplicationControllers {
			interfaces = append(interfaces, NewReplicationControllerController(replicationController))
		}
	}
	if len(interfaces) > 0 {
		return interfaces, nil
	}
	return nil, fmt.Errorf("Controller type (%s) does not have a generator", controllerKind)
}
