package httpclient

const (
	defaultSize = 20
)

type PoolConfig struct {
	// Size - size of pool httpclients
	Size int
}

func (pc *PoolConfig) Initilize() error {
	if pc.Size == 0 {
		pc.Size = defaultSize
	}

	return nil
}
