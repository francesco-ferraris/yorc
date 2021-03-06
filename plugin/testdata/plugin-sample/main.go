package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/plugin"
	"github.com/ystia/yorc/v4/prov"
)

type myDelegateExecutor struct{}

func (d *myDelegateExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	log.Printf("Hello from myDelegateExecutor")

	locationMgr, err := locations.GetManager(cfg)
	if err != nil {
		return err
	}

	props, err := locationMgr.GetLocationProperties("plugin", "locationType")
	if err != nil {
		for _, k := range props.Keys() {
			log.Printf("configuration key: %s", k)
		}
		log.Printf("Secret key: %q", props.GetStringOrDefault("test", "not found!"))
	}

	events.SimpleLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("Hello from myDelegateExecutor")
	return nil
}

type myOperationExecutor struct{}

func (d *myOperationExecutor) ExecAsyncOperation(ctx context.Context, conf config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, stepName string) (*prov.Action, time.Duration, error) {
	return nil, 0, fmt.Errorf("asynchronous operations %v not yet supported by this sample", operation)
}

func (d *myOperationExecutor) ExecOperation(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {
	log.Printf("Hello from myOperationExecutor")
	
	locationMgr, err := locations.GetManager(cfg)
	if err != nil {
		return err
	}

	props, err := locationMgr.GetLocationProperties("plugin", "locationType")
	if err != nil {
		for _, k := range props.Keys() {
			log.Printf("configuration key: %s", k)
		}
		log.Printf("Secret key: %q", props.GetStringOrDefault("test", "not found!"))
	}

	events.SimpleLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("Hello from myOperationExecutor")
	return nil
}

func main() {
	def := []byte(`tosca_definitions_version: yorc_tosca_simple_yaml_1_0

metadata:
  template_name: yorc-my-types
  template_author: Yorc
  template_version: 1.0.0

imports:
  - yorc: <yorc-types.yml>

artifact_types:
  yorc.artifacts.Implementation.MyImplementation:
    derived_from: tosca.artifacts.Implementation
    description: My dummy implementation artifact
    file_ext: [ "myext" ]

node_types:
  yorc.my.types.Compute:
    derived_from: tosca.nodes.Compute

  yorc.my.types.Soft:
    derived_from: tosca.nodes.SoftwareComponent
    interfaces:
      Standard:
        create: dothis.myext

`)

	plugin.Serve(&plugin.ServeOpts{
		DelegateFunc: func() prov.DelegateExecutor {
			return new(myDelegateExecutor)
		},
		DelegateSupportedTypes: []string{`yorc\.my\.types\..*`},
		Definitions: map[string][]byte{
			"my-def.yaml": def,
		},
		OperationFunc: func() prov.OperationExecutor {
			return new(myOperationExecutor)
		},
		OperationSupportedArtifactTypes: []string{"yorc.artifacts.Implementation.MyImplementation"},
	})
}
