package exporter

// http constants
const (
	// apiKeyAuthName contains the name of the header key to authenticate against the exporter API without basic auth.
	apiKeyAuthName = "X-CES-EXPORTER-API-KEY"
)

// exporter endpoints
const (
	// EndpointExportMode contains the endpoint which returns data on the readiness of the exporter system.
	endpointExportMode = "/export/mode"
	// EndpointSystemInfo contains the endpoint which returns data which describe the exporter system, f. i. installed dogus etc.
	endpointSystemInfo = "/system-info"
	// EndpointExportDogu contains the endpoint for getting the current export dogu or setting a new export dogu
	endpointExportDogu = "/export/dogu"
)

// ExportMode contains data about the export readiness of the exporter system.
type ExportMode struct {
	// IsActive indicates whether the exporter system is ready to conduct an export (true) or not (false).
	IsActive bool `json:"isActive"`
}

// Dogu contains data on a single installed dogu on the exporter side.
type Dogu struct {
	// Name holds the dogu name including the namespace delimited by a slash ("/").
	Name string `json:"name"`
	// Version holds the dogu version.
	Version string `json:"version"`
	// Volume contains data on the dogu's persistent storage.
	Volume DoguVolume `json:"volume"`
}

// DoguVolume contains data on the dogu's persistent storage.
type DoguVolume struct {
	// SizeInBytes contains the expected dogu volume size.
	//
	// While int32 (~2 Gibi bytes) is too small for comfort, int64 (~9 Exbi bytes) should suffice to accommodate the
	// size of even the largest dogu volume.
	SizeInBytes int64 `json:"sizeInBytes"`
}

type Component struct {
	// Name holds the component name.
	Name string `json:"name"`
	// Version holds the component version.
	Version string `json:"version"`
}

// SystemInfo contains data on vital data on the exporter side.
type SystemInfo struct {
	// FQDN contains the DNS name of the exporter systeḿ.
	FQDN string `json:"fqdn"`
	// IsMultinode indicates whether the exporter system is a classic CES or a multinode CES instance.
	IsMultinode bool `json:"isMultinode"`
	// Dogus contains data on all installed dogus on the exporter side.
	Dogus []Dogu `json:"dogus"`
	// Components contain data on all installed components on the exporter side.
	Components []Component `json:"components"`
}

type DoguExport struct {
	Dogu         string `json:"dogu"`
	VolumePath   string `json:"volumePath"`
	ExporterPort int    `json:"exporterPort"`
}
