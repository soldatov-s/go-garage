package meta

// ApplicationInfo contains information about service for output in swagger,
// logs and help messages
type ApplicationInfo struct {
	Name        string
	Builded     string
	Hash        string
	Version     string
	Description string
}

func (appInfo *ApplicationInfo) GetBuildInfo() string {
	return appInfo.Version + ", builded: " +
		appInfo.Builded + ", hash: " + appInfo.Hash
}
