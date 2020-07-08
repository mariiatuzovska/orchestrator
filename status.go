package orchestrator

import "time"

// Status - general structure for Services, Nodes and orchestrator
type Status struct {
	OK                     bool              // is running
	List                   map[string]string // list of statuses key : value
	Error                  string            // last fatal error or ""
	ThisUpdate, NextUpdate string
}

func NewStatus() *Status {
	return &Status{false, make(map[string]string), "", time.Now().String(), ""}
}

func NewInitializedStatus() *Status {
	m := map[string]string{StatusNameMap[StatusNameGeneral]: StatusMap[StatusInitialized]}
	return &Status{false, m, "", time.Now().String(), ""}
}

func NewStoppedStatus(Error string) *Status {
	m := map[string]string{StatusNameMap[StatusNameGeneral]: StatusMap[StatusStopped]}
	return &Status{false, m, Error, time.Now().String(), ""}
}

func (s *Status) SetListStatus(key, value int) {
	s.List[StatusNameMap[key]] = StatusMap[value]
}
