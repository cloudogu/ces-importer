package systeminfo

import (
	"context"
	"testing"

	"github.com/cloudogu/ces-importer/migration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	t.Run("should return new validator", func(t *testing.T) {
		v := NewValidator([]string{"test1", "test2"})
		require.NotNil(t, v)
		assert.Equal(t, []string{"test1", "test2"}, v.excludedDogus)
	})
}

func TestValidateSystemInfo(t *testing.T) {
	t.Run("should return with no error", func(t *testing.T) {
		sysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []migration.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}

		v := Validator{}
		err := v.Validate(context.Background(), &sysInfo, &sysInfo)
		require.NoError(t, err)
	})

	t.Run("should return error mismatching dogu versions", func(t *testing.T) {
		exsysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []migration.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testdogu",
					Version: "9.9.9",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []migration.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}

		v := Validator{}
		err := v.Validate(context.Background(), &exsysInfo, &imSysInfo)
		require.ErrorContains(t, err, "version discrepancy for dogu testdogu. Source instance version: 1.2.3, Target instance version: 9.9.9")
	})

	t.Run("should return error dogu not installed", func(t *testing.T) {
		exsysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []migration.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{},
			Components: []migration.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}

		v := Validator{}
		err := v.Validate(context.Background(), &exsysInfo, &imSysInfo)
		require.ErrorContains(t, err, "dogu testdogu is not installed (required version: 1.2.3)")
	})

	t.Run("should return error component not installed", func(t *testing.T) {
		exsysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []migration.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []migration.Component{},
		}

		v := Validator{}
		err := v.Validate(context.Background(), &exsysInfo, &imSysInfo)
		require.ErrorContains(t, err, "component testcomponent is not installed (required version: 1.2.3)")
	})

	t.Run("should return error component mismatching component version", func(t *testing.T) {
		exsysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []migration.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		imSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []migration.Component{
				{
					Name:    "testcomponent",
					Version: "9.9.9",
				},
			},
		}

		v := Validator{}
		err := v.Validate(context.Background(), &exsysInfo, &imSysInfo)
		require.ErrorContains(t, err, "version discrepancy for component testcomponent. Source instance version: 1.2.3, Target instance version: 9.9.9")
	})

	t.Run("should error on dogu not installed in exporting system", func(t *testing.T) {
		imSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
				{
					Name:    "onlyPresentHere",
					Version: "1.2.3",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []migration.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}
		exSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testdogu",
					Version: "1.2.3",
					Volume: migration.DoguVolume{
						SizeInBytes: 10,
					},
				},
			},
			Components: []migration.Component{
				{
					Name:    "testcomponent",
					Version: "1.2.3",
				},
			},
		}

		v := Validator{}
		err := v.Validate(context.Background(), &exSysInfo, &imSysInfo)
		require.ErrorContains(t, err, "dogu onlyPresentHere is installed in the importing system but not present in the exporting system")
	})

	t.Run("should throw no error on version mismatch for excluded dogu that is present on both exporting and importing system", func(t *testing.T) {
		imSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "official/excludeddogu",
					Version: "1.2.3",
				},
			},
		}
		exSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "official/excludeddogu",
					Version: "1.2.5",
				},
			},
		}

		v := Validator{
			excludedDogus: []string{"excludeddogu"},
		}
		err := v.Validate(context.Background(), &exSysInfo, &imSysInfo)
		require.NoError(t, err)

	})
	t.Run("should throw no error on version mismatch for excluded dogu that is present on both exporting and importing system with different namespaces", func(t *testing.T) {
		imSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "official/excludeddogu",
					Version: "1.2.3",
				},
			},
		}
		exSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testing/excludeddogu",
					Version: "1.2.5",
				},
			},
		}

		v := Validator{
			excludedDogus: []string{"excludeddogu"},
		}
		err := v.Validate(context.Background(), &exSysInfo, &imSysInfo)
		require.NoError(t, err)

	})

	t.Run("should throw no error on excluded dogu not installed in the importing system", func(t *testing.T) {
		imSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{},
		}
		exSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "testing/excludeddogu",
					Version: "1.2.5",
				},
			},
		}

		v := Validator{
			excludedDogus: []string{"excludeddogu"},
		}
		err := v.Validate(context.Background(), &exSysInfo, &imSysInfo)
		require.NoError(t, err)

	})
	t.Run("should throw no error on excluded dogu not installed in the exporting system", func(t *testing.T) {
		imSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{
				{
					Name:    "official/excludeddogu",
					Version: "1.2.3",
				},
			},
		}
		exSysInfo := migration.SystemInfo{
			Dogus: []migration.Dogu{},
		}

		v := Validator{
			excludedDogus: []string{"excludeddogu"},
		}
		err := v.Validate(context.Background(), &exSysInfo, &imSysInfo)
		require.NoError(t, err)

	})

}
