package domain

type MachineStatus string

const (
	MachineStatusOnline       MachineStatus = "online"
	MachineStatusOffline      MachineStatus = "offline"
	MachineStatusReconnecting MachineStatus = "reconnecting"
)

type Machine struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Status MachineStatus `json:"status"`
}
