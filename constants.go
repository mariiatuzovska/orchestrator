package orchestrator

const (
	StatusInitialized StatusValue = "Initialized"
	StatusRunning     StatusValue = "Running"
	StatusStopped     StatusValue = "Stopped"
	StatusFailed      StatusValue = "Failed"
	StatusPassed      StatusValue = "Passed"
	StatusUnknown     StatusValue = "Unknown"

	SettingValueNever     SettingValue = "Never"
	SettingValueNow       SettingValue = "Now"
	SettingValueOnFailure SettingValue = "OnFailure"
)

const (
	OSLinux   = "linux"
	OSWindows = "windows"
	OSDarwin  = "darwin"
)

var HttpMethodMap = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"PATCH":  true,
	"DELETE": true,
}

var SettingValueMap = map[SettingValue]bool{
	SettingValueNever:     true,
	SettingValueNow:       true,
	SettingValueOnFailure: true,
}

var DefaultTimeout = 300 // seconds
