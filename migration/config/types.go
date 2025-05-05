package configuration

type keyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type globalConfig []keyValue

type doguConfig struct {
	Name            string     `json:"name"`
	NormalConfig    []keyValue `json:"normal"`
	LocalConfig     []keyValue `json:"local"`
	SensitiveConfig []keyValue `json:"sensitive"`
}

type backupSchedule struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
}

type configuration struct {
	GlobalConfig    globalConfig     `json:"global"`
	DoguConfigs     []doguConfig     `json:"dogus"`
	BackupSchedules []backupSchedule `json:"backupSchedules"`
}
