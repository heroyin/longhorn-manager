package api

import (
	"strconv"

	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"

	"github.com/yasker/lm-rewrite/manager"
	"github.com/yasker/lm-rewrite/types"
)

type Volume struct {
	client.Resource

	Name                string `json:"name,omitempty"`
	Size                string `json:"size,omitempty"`
	BaseImage           string `json:"baseImage,omitempty"`
	FromBackup          string `json:"fromBackup,omitempty"`
	NumberOfReplicas    int    `json:"numberOfReplicas,omitempty"`
	StaleReplicaTimeout int    `json:"staleReplicaTimeout,omitempty"`
	State               string `json:"state,omitempty"`
	EngineImage         string `json:"engineImage,omitempty"`
	Endpoint            string `json:"endpoint,omitemtpy"`
	Created             string `json:"created,omitemtpy"`

	Replicas   []Replica   `json:"replicas,omitempty"`
	Controller *Controller `json:"controller,omitempty"`
}

type Host struct {
	client.Resource

	UUID    string `json:"uuid,omitempty"`
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
}

type Instance struct {
	Name    string `json:"name,omitempty"`
	NodeID  string `json:"hostId,omitempty"`
	Address string `json:"address,omitempty"`
	Running bool   `json:"running,omitempty"`
}

type Controller struct {
	Instance
}

type Replica struct {
	Instance

	Mode     string `json:"mode,omitempty"`
	FailedAt string `json:"badTimestamp,omitempty"`
}

type AttachInput struct {
	HostID string `json:"hostId,omitempty"`
}

type Empty struct{}

func NewSchema() *client.Schemas {
	schemas := &client.Schemas{}

	schemas.AddType("apiVersion", client.Resource{})
	schemas.AddType("schema", client.Schema{})
	schemas.AddType("error", client.ServerApiError{})
	schemas.AddType("attachInput", AttachInput{})

	hostSchema(schemas.AddType("host", Host{}))
	volumeSchema(schemas.AddType("volume", Volume{}))

	return schemas
}

func hostSchema(host *client.Schema) {
	host.CollectionMethods = []string{"GET"}
	host.ResourceMethods = []string{"GET"}
}

func volumeSchema(volume *client.Schema) {
	volume.CollectionMethods = []string{"GET", "POST"}
	volume.ResourceMethods = []string{"GET", "DELETE"}
	volume.ResourceActions = map[string]client.Action{
		"attach": {
			Input:  "attachInput",
			Output: "volume",
		},
		"detach": {
			Output: "volume",
		},
	}
	volume.ResourceFields["controller"] = client.Field{
		Type:     "struct",
		Nullable: true,
	}
	volumeName := volume.ResourceFields["name"]
	volumeName.Create = true
	volumeName.Required = true
	volumeName.Unique = true
	volume.ResourceFields["name"] = volumeName

	volumeSize := volume.ResourceFields["size"]
	volumeSize.Create = true
	volumeSize.Required = true
	volumeSize.Default = "100G"
	volume.ResourceFields["size"] = volumeSize

	volumeFromBackup := volume.ResourceFields["fromBackup"]
	volumeFromBackup.Create = true
	volume.ResourceFields["fromBackup"] = volumeFromBackup

	volumeNumberOfReplicas := volume.ResourceFields["numberOfReplicas"]
	volumeNumberOfReplicas.Create = true
	volumeNumberOfReplicas.Required = true
	volumeNumberOfReplicas.Default = 2
	volume.ResourceFields["numberOfReplicas"] = volumeNumberOfReplicas

	volumeStaleReplicaTimeout := volume.ResourceFields["staleReplicaTimeout"]
	volumeStaleReplicaTimeout.Create = true
	volumeStaleReplicaTimeout.Default = 20
	volume.ResourceFields["staleReplicaTimeout"] = volumeStaleReplicaTimeout
}

func toVolumeResource(v *types.VolumeInfo, vc *types.ControllerInfo, vrs map[string]*types.ReplicaInfo, apiContext *api.ApiContext) *Volume {
	replicas := []Replica{}
	for _, r := range vrs {
		/*
			mode := ""
			if r.Running {
				mode = string(r.Mode)
			}
		*/
		replicas = append(replicas, Replica{
			Instance: Instance{
				Name:    r.Name,
				Running: r.Running,
				Address: r.IP,
				NodeID:  r.NodeID,
			},
			//Mode:     mode,
			FailedAt: r.FailedAt,
		})
	}

	var controller *Controller
	if vc != nil {
		controller = &Controller{Instance{
			Name:    vc.Name,
			Running: vc.Running,
			NodeID:  vc.NodeID,
			Address: vc.IP,
		}}
	}

	r := &Volume{
		Resource: client.Resource{
			Id:      v.Name,
			Type:    "volume",
			Actions: map[string]string{},
			Links:   map[string]string{},
		},
		Name:             v.Name,
		Size:             strconv.FormatInt(v.Size, 10),
		BaseImage:        v.BaseImage,
		FromBackup:       v.FromBackup,
		NumberOfReplicas: v.NumberOfReplicas,
		State:            string(v.State),
		//EngineImage:         v.EngineImage,
		//RecurringJobs:       v.RecurringJobs,
		StaleReplicaTimeout: v.StaleReplicaTimeout,
		Endpoint:            v.Endpoint,
		Created:             v.Created,

		Controller: controller,
		Replicas:   replicas,
	}

	actions := map[string]struct{}{}

	switch v.State {
	case types.VolumeStateDetached:
		actions["attach"] = struct{}{}
		//actions["recurringUpdate"] = struct{}{}
		//actions["replicaRemove"] = struct{}{}
	case types.VolumeStateHealthy:
		actions["detach"] = struct{}{}
		//actions["snapshotPurge"] = struct{}{}
		//actions["snapshotCreate"] = struct{}{}
		//actions["snapshotList"] = struct{}{}
		//actions["snapshotGet"] = struct{}{}
		//actions["snapshotDelete"] = struct{}{}
		//actions["snapshotRevert"] = struct{}{}
		//actions["snapshotBackup"] = struct{}{}
		//actions["recurringUpdate"] = struct{}{}
		//actions["bgTaskQueue"] = struct{}{}
		//actions["replicaRemove"] = struct{}{}
	case types.VolumeStateDegraded:
		actions["detach"] = struct{}{}
		//actions["snapshotPurge"] = struct{}{}
		//actions["snapshotCreate"] = struct{}{}
		//actions["snapshotList"] = struct{}{}
		//actions["snapshotGet"] = struct{}{}
		//actions["snapshotDelete"] = struct{}{}
		//actions["snapshotRevert"] = struct{}{}
		//actions["snapshotBackup"] = struct{}{}
		//actions["recurringUpdate"] = struct{}{}
		//actions["bgTaskQueue"] = struct{}{}
		//actions["replicaRemove"] = struct{}{}
	case types.VolumeStateCreated:
		actions["recurringUpdate"] = struct{}{}
	case types.VolumeStateFault:
	}

	for action := range actions {
		r.Actions[action] = apiContext.UrlBuilder.ActionLink(r.Resource, action)
	}

	return r
}

func toHostCollection(hosts map[string]*types.NodeInfo) *client.GenericCollection {
	data := []interface{}{}
	for _, v := range hosts {
		data = append(data, toHostResource(v))
	}
	return &client.GenericCollection{Data: data}
}

func toHostResource(h *types.NodeInfo) *Host {
	return &Host{
		Resource: client.Resource{
			Id:      h.ID,
			Type:    "host",
			Actions: map[string]string{},
		},
		UUID:    h.ID,
		Name:    h.Name,
		Address: h.IP,
	}
}

type Server struct {
	m *manager.VolumeManager
}

func NewServer(m *manager.VolumeManager) *Server {
	return &Server{
		m: m,
	}
}