package orchestrator

// STATUSES
const (
	StatusInitialized   = -1
	StatusActive        = 0
	StatusConnected     = 0
	StatusPassed        = 0
	StatusInactive      = 1
	StatusDisconnected  = 1
	StatusFailed        = 1
	StatusNilConnection = 2
	StatusUnknownNode   = 2
	StatusUnknownOS     = 2
	StatusUndefined     = -2
)

const (
	OrchestratorServiceType = "orchestrator"

	OSLinux   = "linux"
	OSWindows = "windows"
	OSDarwin  = "darwin"

	NameOfThisNode = "this"

	LinuxTryIsActiveFormatString   = "systemctl is-active %s --quiet; echo $?" // + ServiceConfiguration.ServiceName
	DarwinTryIsActiveFormatString  = "launchctl list | grep %s"                // + ServiceConfiguration.ServiceName
	WindowsTryIsActiveFormatString = ""

	LinuxStartServiceFormatString  = "systemctl start %s" // + ServiceConfiguration.ServiceName
	DarwinStartServiceFormatString = "launchctl load %s"  // + ServiceConfiguration.ServiceName

	LinuxStopServiceFormatString  = "systemctl stop %s"   // + ServiceConfiguration.ServiceName
	DarwinStopServiceFormatString = "launchctl unload %s" // + ServiceConfiguration.ServiceName
)

var HttpMethodMap = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"PATCH":  true,
	"DELETE": true,
}
