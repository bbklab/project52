package main

import (
	"encoding/json"
	"errors"

	"./mesosproto"

	"github.com/gogo/protobuf/proto"
)

func newTask(taskID, agentID string) *mesosproto.TaskInfo {
	return &mesosproto.TaskInfo{
		Name: &taskID,
		TaskId: &mesosproto.TaskID{
			Value: &taskID,
		},
		AgentId: &mesosproto.AgentID{
			Value: &agentID,
		},
		Resources: []*mesosproto.Resource{
			&mesosproto.Resource{
				Name: proto.String("cpus"),
				Type: mesosproto.Value_SCALAR.Enum(),
				Scalar: &mesosproto.Value_Scalar{
					Value: proto.Float64(0.001),
				},
			},
			&mesosproto.Resource{
				Name: proto.String("mem"),
				Type: mesosproto.Value_SCALAR.Enum(),
				Scalar: &mesosproto.Value_Scalar{
					Value: proto.Float64(1),
				},
			},
		},
		Command: &mesosproto.CommandInfo{
			Shell:     proto.Bool(false),
			Value:     proto.String("sleep"),
			Arguments: []string{"1000d"},
		},
		Container: &mesosproto.ContainerInfo{
			Type: mesosproto.ContainerInfo_DOCKER.Enum(),
			Docker: &mesosproto.ContainerInfo_DockerInfo{
				Image:          proto.String(image),
				Network:        mesosproto.ContainerInfo_DockerInfo_BRIDGE.Enum(),
				Privileged:     proto.Bool(false),
				ForcePullImage: proto.Bool(false),
			},
		},
	}
}

func IsTaskDone(status *mesosproto.TaskStatus) bool {
	state := status.GetState()

	switch state {
	case mesosproto.TaskState_TASK_RUNNING,
		mesosproto.TaskState_TASK_FINISHED,
		mesosproto.TaskState_TASK_FAILED,
		mesosproto.TaskState_TASK_KILLED,
		mesosproto.TaskState_TASK_ERROR,
		mesosproto.TaskState_TASK_LOST,
		mesosproto.TaskState_TASK_DROPPED,
		mesosproto.TaskState_TASK_GONE:
		return true
	}

	return false
}

func DetectError(status *mesosproto.TaskStatus) error {
	var (
		state = status.GetState()
		//data  = status.GetData() // docker container inspect result
	)

	switch state {
	case mesosproto.TaskState_TASK_FAILED,
		mesosproto.TaskState_TASK_ERROR,
		mesosproto.TaskState_TASK_LOST,
		mesosproto.TaskState_TASK_DROPPED,
		mesosproto.TaskState_TASK_UNREACHABLE,
		mesosproto.TaskState_TASK_GONE,
		mesosproto.TaskState_TASK_GONE_BY_OPERATOR,
		mesosproto.TaskState_TASK_UNKNOWN:
		bs, _ := json.Marshal(map[string]interface{}{
			"state":   state.String(),
			"message": status.GetMessage(),
			"source":  status.GetSource().String(),
			"reason":  status.GetReason().String(),
			"healthy": status.GetHealthy(),
		})
		return errors.New(string(bs))
	}

	return nil
}
