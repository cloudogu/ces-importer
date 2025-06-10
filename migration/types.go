package migration

// Dogu contains data on a single installed dogu on the exporter side.
type Dogu struct {
	// Name holds the dogu name including the namespace delimited by a slash ("/").
	Name string
	// Version holds the dogu version.
	Version string
	// Volume contains data on the dogu's persistent storage.
	Volume DoguVolume
}

// DoguVolume contains data on the dogu's persistent storage.
type DoguVolume struct {
	// SizeInBytes contains the expected dogu volume size.
	//
	// While int32 (~2 Gibi bytes) is too small for comfort, int64 (~9 Exbi bytes) should suffice to accommodate the
	// size of even the largest dogu volume.
	SizeInBytes int64
}

type Component struct {
	// Name holds the component name.
	Name string
	// Version holds the component version.
	Version string
}

// SystemInfo contains data on vital data on the exporter side.
type SystemInfo struct {
	// FQDN contains the DNS name of the exporter systeḿ.
	FQDN string
	// IsMultinode indicates whether the exporter system is a classic CES or a multinode CES instance.
	IsMultinode bool
	// Dogus contains data on all installed dogus on the exporter side.
	Dogus []Dogu
	// Components contain data on all installed components on the exporter side.
	Components []Component
}

// DoguExport contains data for exporting a Dogu, including its name, volume path, and exporter port configuration.
type DoguExport struct {
	// Dogu specifies the name of the Dogu to be exported.
	Dogu string
	// VolumePath specifies the file system path to the volume associated with the Dogu being exported.
	VolumePath string
	// ExporterPort defines the network port used by the exporter process for transferring data.
	ExporterPort int
}

// KeyValue contains a key-value pair.
// This is used to represent configuration values.
type KeyValue struct {
	// Key is the name of the configuration value.
	Key string
	// Value is the value of the configuration value.
	Value string
}

// GlobalConfig contains the global configuration of the exporter system.
type GlobalConfig []KeyValue

// DoguConfig contains the configuration of a single dogu.
type DoguConfig struct {
	// Name of the dogu.
	Name string
	// NormalConfig normal dogu configuration as KeyValue pairs
	NormalConfig []KeyValue
	// LocalConfig local dogu configuration as KeyValue pairs
	LocalConfig []KeyValue
	// SensitiveConfig sensitive dogu configuration as KeyValue pairs
	SensitiveConfig []KeyValue
}

// BackupSchedule contains the configuration of a single backup schedule.
type BackupSchedule struct {
	// Name of the backup schedule.
	Name string
	// Schedule is defined as a cron expression.
	Schedule string
}

// Configuration contains the configuration of the exporter system.
type Configuration struct {
	// GlobalConfig is the global configuration of the exporter system.
	GlobalConfig GlobalConfig
	// DoguConfigs is the configuration of all installed dogus in the exporter system.
	DoguConfigs []DoguConfig
	// BackupSchedules is the configuration of all backup schedules in the exporter system.
	BackupSchedules []BackupSchedule
}
