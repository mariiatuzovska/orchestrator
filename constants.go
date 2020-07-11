package orchestrator

const (
	OSLinux   = "linux"
	OSWindows = "windows"
	OSDarwin  = "darwin"

	NameOfThisNode = "this"

	LinuxTryIsActiveFormatString   = "systemctl is-active %s --quiet; echo $?" // + ServiceConfiguration.Name
	DarwinTryIsActiveFormatString  = "launchctl status | grep %s"              // + ServiceConfiguration.Name
	WindowsTryIsActiveFormatString = ""

	LinuxStartServiceFormatString  = "systemctl start %s" // + ServiceConfiguration.Name
	DarwinStartServiceFormatString = "launchctl load %s"  // + ServiceConfiguration.Name

	LinuxStopServiceFormatString  = "systemctl stop %s"   // + ServiceConfiguration.Name
	DarwinStopServiceFormatString = "launchctl unload %s" // + ServiceConfiguration.Name

	StatusActive   = "active"
	StatusInActive = "in-active"
)

var HttpMethodMap = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"PATCH":  true,
	"DELETE": true,
}
