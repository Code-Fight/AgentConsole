package domain

type MachineStatus string
type MachineRuntimeStatus string

const (
	MachineStatusOnline       MachineStatus = "online"
	MachineStatusOffline      MachineStatus = "offline"
	MachineStatusReconnecting MachineStatus = "reconnecting"

	MachineRuntimeStatusUnknown MachineRuntimeStatus = "unknown"
	MachineRuntimeStatusRunning MachineRuntimeStatus = "running"
	MachineRuntimeStatusStopped MachineRuntimeStatus = "stopped"
)

type Machine struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	Status        MachineStatus        `json:"status"`
	RuntimeStatus MachineRuntimeStatus `json:"runtimeStatus,omitempty"`
}
