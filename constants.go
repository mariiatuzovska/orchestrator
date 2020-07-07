package orchestartor

const (
	StatusInitialized = iota
	StatusRunning
	StatusStopped
	StatusUnknown
	StatusHTTPAccesMethodFailed
	StatusGetStatusFailed
	StatusGetIsActiveFailed
)

const (
	OSLinux = iota
	OSWindows
	OSDarwin
)

const (
	TypeOrchestrator = iota
)

const (
	SettingsNever = iota
	SettingsNow
	SettingsOnFailure
)

var StatusNameMap = map[int]string{
	StatusInitialized:           "Initialized",
	StatusRunning:               "Running",
	StatusStopped:               "Stoped",
	StatusUnknown:               "Unknown",
	StatusHTTPAccesMethodFailed: "HTTP access method failed",
	StatusGetIsActiveFailed:     "IsActive command failed",
	StatusGetStatusFailed:       "Status command failed",
}

var HttpMethodMap = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"PATCH":  true,
	"DELETE": true,
}

var SettingsMap = map[string]int{
	"never":      SettingsNever,
	"on-failure": SettingsOnFailure,
	"now":        SettingsNow,
}

var DefaultTimeout = 300 // seconds
