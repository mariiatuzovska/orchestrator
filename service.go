package orchestartor

type Service struct {
	DNS   string
	Nodes []*Node
	// ServiceType ServiceType
}

// ServiceName is a unique value for orchestrators configuration file.
// Defines service name
type ServiceName string

// // ServiceType is a unique value for orchestrators configuration file.
// // Defines service type
// type ServiceType int

// func (s *Service) Valid() bool {
// 	return true
// }

// func (s *Service) Status() *Status {
// 	return nil
// }

// func (s *Service) Start() error {
// 	return nil
// }

// func (s *Service) Stop() error {
// 	return nil
// }
