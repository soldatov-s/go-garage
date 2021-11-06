package config

// import (
// 	//
// 	"testing"

//
// 	"github.com/stretchr/testify/require"
// )

// const (
// 	testEnvParserProviderName = "environment"
// )

// // DummyProvider is a dummy configuration provider.
// type DummyProvider struct{}

// func (da *DummyProvider) Initialize()                       {}
// func (da *DummyProvider) Parse(structure interface{}) error { return nil }

// // EnvParserProvider is a simple configuration provider that parses
// // environment into structure fields.
// type EnvParserProvider struct{}

// func (epp *EnvParserProvider) Initialize() {}

// func (epp *EnvParserProvider) Parse(structure interface{}) error {
// 	structure.(*configstruct).TestVar = "TEST"
// 	structure.(*configstruct).AnotherTestVar = "ANOTHERTEST"
// 	return nil
// }

// // This is a test configuration structure.
// type configstruct struct {
// 	TestVar        string `envconfig:"default=TEST"`
// 	AnotherTestVar string
// }

// func (c *configstruct) FillGowork(config *Configuration) error {
// 	return nil
// }

// func TestConfigurationInitialize(t *testing.T) {
// 	cfgstruct := &configstruct{}
// 	cf := NewConfiguration(cfgstruct)
// 	require.NotNil(t, cf.providers)
// }

// func TestConfigurationParse(t *testing.T) {
// 	cfgstruct := &configstruct{}
// 	cf := NewConfiguration(cfgstruct)
// 	require.NotNil(t, cf.providers)

// 	p := &EnvParserProvider{}
// 	_ = cf.RegisterProvider(testEnvParserProviderName, p)

// 	cf.ProvidersOrder = make([]string, 0)
// 	cf.ProvidersOrder = append(cf.ProvidersOrder, testEnvParserProviderName)

// 	err := cf.Parse()
// 	require.Nil(t, err)

// 	testOnPanic := func() {
// 		_ = cf.Service.(*configstruct)
// 	}

// 	require.NotPanics(t, testOnPanic)

// 	require.Equal(t, "TEST", cf.Service.(*configstruct).TestVar)
// 	require.Equal(t, "ANOTHERTEST", cf.Service.(*configstruct).AnotherTestVar)
// }

// func TestConfigurationRegisterProvider(t *testing.T) {
// 	cfgstruct := &configstruct{}
// 	cf := NewConfiguration(cfgstruct)
// 	require.NotNil(t, cf.providers)

// 	d := &DummyProvider{}
// 	err := cf.RegisterProvider("dummy", d)
// 	require.Nil(t, err)
// 	require.Len(t, cf.providers, 1)
// 	require.Contains(t, cf.providers, "dummy")
// }

// func TestConfigurationRegisterAlreadyRegisteredProvider(t *testing.T) {
// 	cfgstruct := &configstruct{}
// 	cf := NewConfiguration(cfgstruct)
// 	require.NotNil(t, cf.providers)

// 	d := &DummyProvider{}
// 	err := cf.RegisterProvider("dummy", d)
// 	require.Nil(t, err)
// 	require.Len(t, cf.providers, 1)
// 	require.Contains(t, cf.providers, "dummy")

// 	dd := &DummyProvider{}
// 	err1 := cf.RegisterProvider("dummy", dd)
// 	require.NotNil(t, err1)
// 	require.Len(t, cf.providers, 1)
// 	require.Contains(t, cf.providers, "dummy")
// }
