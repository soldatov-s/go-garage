package domains

type IBaseDomainStarter interface {
	Start() error
}

type IBaseDomainShutdowner interface {
	Shutdown() error
}

type IBaseDomain interface {
	IBaseDomainStarter
	IBaseDomainShutdowner
}
