package orchestrator

import "time"

// Status - general structure for Services, Nodes and orchestrator
type Status struct {
	GeneralStatus          StatusValue // Initialized / Running / Stopped / Failed
	StatusList             StatusList  // list of statuses CommandName : StatusValue
	Error                  string      // last fatal error or ""
	ThisUpdate, NextUpdate string
}

type StatusValue string

type StatusList map[CommandName]StatusValue

func NewStatusInitialized() *Status {
	return &Status{StatusInitialized, make(StatusList), "", time.Now().String(), ""}
}

func NewStatusRunning() *Status {
	return &Status{StatusRunning, make(StatusList), "", time.Now().String(), ""}
}

func NewStatusStopped(Error string) *Status {
	return &Status{StatusStopped, make(StatusList), Error, time.Now().String(), ""}
}

func NewStatusFailed(Error string) *Status {
	return &Status{StatusFailed, make(StatusList), Error, time.Now().String(), ""}
}

func (s *Status) SetListStatus(key CommandName, value StatusValue) {
	s.StatusList[key] = value
}
