package orchestrator

const (
	StatusInitialized = iota
	StatusRunning
	StatusStopped
	StatusUnknown
	StatusFailed
	StatusOK
	// Names
	StatusNameGeneral
	StatusNameHTTPAccess
	StatusNameStatus
	StausNameIsActive
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
	StatusNameGeneral:    "General",
	StatusNameHTTPAccess: "HTTPAccess",
	StatusNameStatus:     "Status",
	StausNameIsActive:    "IsActive",
}

var StatusMap = map[int]string{
	StatusInitialized: "Initialized",
	StatusRunning:     "Running",
	StatusStopped:     "Stoped",
	StatusUnknown:     "Unknown",
	StatusFailed:      "Failed",
	StatusOK:          "OK",
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
