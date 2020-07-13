package orchestrator

const (
	OrchestratorServiceType = "orchestrator"

	OSLinux   = "linux"
	OSWindows = "windows"
	OSDarwin  = "darwin"

	NameOfThisNode = "this"

	LinuxTryIsActiveFormatString   = "systemctl is-active %s --quiet; echo $?" // + ServiceConfiguration.Name
	DarwinTryIsActiveFormatString  = "launchctl list | grep %s"                // + ServiceConfiguration.Name
	WindowsTryIsActiveFormatString = ""

	LinuxStartServiceFormatString  = "systemctl start %s" // + ServiceConfiguration.Name
	DarwinStartServiceFormatString = "launchctl load %s"  // + ServiceConfiguration.Name

	LinuxStopServiceFormatString  = "systemctl stop %s"   // + ServiceConfiguration.Name
	DarwinStopServiceFormatString = "launchctl unload %s" // + ServiceConfiguration.Name

	StatusActive   = "active"
	StatusInactive = "inactive"

	HTTPAccessStatusUndefined = "undefined"
	HTTPAccessStatusPassed    = "passed"
)

var HttpMethodMap = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"PATCH":  true,
	"DELETE": true,
}
