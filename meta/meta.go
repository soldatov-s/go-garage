package meta

type Config struct {
	Name        string
	Builded     string
	Hash        string
	Version     string
	Description string
}

// AppInfo contains information about service for output in swagger,
// logs and help messages
type AppInfo struct {
	Name        string
	Builded     string
	Hash        string
	Version     string
	Description string
}

func DefaultAppInfo() *AppInfo {
	return &AppInfo{
		Name:        "unknown",
		Version:     "0.0.0",
		Description: "no description",
	}
}

func NewAppInfo(cfg *Config) *AppInfo {
	return &AppInfo{
		Name:        cfg.Name,
		Builded:     cfg.Name,
		Hash:        cfg.Hash,
		Version:     cfg.Version,
		Description: cfg.Description,
	}
}

func (appInfo *AppInfo) GetBuildInfo() string {
	return appInfo.Version + ", builded: " + appInfo.Builded + ", hash: " + appInfo.Hash
}
