package orchestrator

// STATUSES
const (
	StatusInitialized   = -1
	StatusUndefined     = -2
	StatusActive        = 0
	StatusConnected     = 0
	StatusPassed        = 0
	StatusInactive      = 1
	StatusDisconnected  = 1
	StatusFailed        = 0x200 // http access failed
	StatusNilConnection = 0x400 // connection must be used, but it is null
	StatusUnknownNode   = 0x501 // node is not found by name
	StatusUnknownOS     = 0x502 // undefined OS
)

const (
	Debug = iota
	Info
	Warning
	Error
	Fatal
)

const (
	OrchestratorServiceType = "orchestrator"

	OSLinux   = "linux"
	OSWindows = "windows"
	OSDarwin  = "darwin"

	LinuxTryIsActiveFormatString  = "systemctl is-active %s --quiet; echo $?"   // + ServiceConfiguration.ServiceName
	DarwinTryIsActiveFormatString = "launchctl list | grep %s --quiet; echo $?" // + ServiceConfiguration.ServiceName

	LinuxStartServiceFormatString  = "systemctl start %s" // + ServiceConfiguration.ServiceName
	DarwinStartServiceFormatString = "launchctl start %s" // + ServiceConfiguration.ServiceName

	LinuxStopServiceFormatString  = "systemctl stop %s" // + ServiceConfiguration.ServiceName
	DarwinStopServiceFormatString = "launchctl stop %s" // + ServiceConfiguration.ServiceName

	LinuxInstallingDebFormatString = "dpkg -i %s" // + ServiceTemplate.ServioceName
)

var HttpMethodMap = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"PATCH":  true,
	"DELETE": true,
}
