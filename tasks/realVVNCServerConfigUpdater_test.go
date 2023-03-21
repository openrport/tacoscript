//go:build !windows
// +build !windows

package tasks

import (
	"context"
	"io/fs"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/tacoscript/utils"
)

const (
	origTestConfigFilename = "../realvnc/test/realvncserver-config.conf.orig"
	testConfigFilename     = "../realvnc/test/realvncserver-config.conf"
)

func testSetup(t *testing.T) {
	t.Helper()
	makeTestConfigFile(t)
}

func testTeardown(t *testing.T) {
	t.Helper()
	removeTestConfigFile(t)
}

func makeTestConfigFile(t *testing.T) {
	contents, err := os.ReadFile(origTestConfigFilename)
	require.NoError(t, err)
	err = os.WriteFile(testConfigFilename, contents, 0644) //nolint:gosec // test file
	require.NoError(t, err)
}

func removeTestConfigFile(t *testing.T) {
	err := os.Remove(testConfigFilename)
	require.NoError(t, err)

	_ = os.Remove(utils.GetBackupFilename(testConfigFilename, "bak"))
}

func TestShouldUpdateSimpleConfigFileParam(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)

	ctx := context.Background()

	executor := &RealVNCServerTaskExecutor{
		FsManager: &utils.FsManager{},

		Reloader: &mockConfigReloader{},
	}

	tracker := &FieldNameStatusTracker{
		nameMap: withNameMap("encryption", "Encryption"),
		statusMap: fieldStatusMap{
			"Encryption": FieldStatus{
				HasNewValue: true,
			},
		},
	}

	task := &RealVNCServerTask{
		Path:       "realvnc-server-1",
		ConfigFile: "../realvnc/test/realvncserver-config.conf",
		Encryption: "AlwaysOn",
		mapper:     tracker,
		tracker:    tracker,
	}

	err := task.Validate(runtime.GOOS)
	require.NoError(t, err)

	res := executor.Execute(ctx, task)
	require.NoError(t, res.Err)
	require.True(t, task.Updated)

	assert.Equal(t, res.Comment, "Config updated")
	assert.Equal(t, res.Changes["count"], "1 config value change(s) applied")

	contents, err := os.ReadFile(task.ConfigFile)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "Encryption=AlwaysOn")

	backupContents, err := os.ReadFile(utils.GetBackupFilename(task.ConfigFile, "bak"))
	require.NoError(t, err)
	assert.Contains(t, string(backupContents), "Encryption=BadValue")
}

func TestShouldAddSimpleConfigFileParam(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)

	ctx := context.Background()

	executor := &RealVNCServerTaskExecutor{
		FsManager: &utils.FsManager{},

		Reloader: &mockConfigReloader{},
	}

	tracker := &FieldNameStatusTracker{
		nameMap: withNameMap("blank_screen", "BlankScreen"),
		statusMap: fieldStatusMap{
			"BlankScreen": FieldStatus{
				HasNewValue: true,
			},
		},
	}

	task := &RealVNCServerTask{
		Path:        "realvnc-server-1",
		ConfigFile:  "../realvnc/test/realvncserver-config.conf",
		SkipBackup:  true,
		BlankScreen: true,
		mapper:      tracker,
		tracker:     tracker,
	}

	err := task.Validate(runtime.GOOS)
	require.NoError(t, err)

	res := executor.Execute(ctx, task)
	require.NoError(t, res.Err)
	require.True(t, task.Updated)

	assert.Equal(t, "Config updated", res.Comment)
	assert.Equal(t, "1 config value change(s) applied", res.Changes["count"])

	contents, err := os.ReadFile(task.ConfigFile)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "BlankScreen=true")

	_, err = os.ReadFile(utils.GetBackupFilename(task.ConfigFile, "bak"))
	require.ErrorContains(t, err, "no such file")
}

func TestShouldAddSimpleConfigWhenNoExistingConfigFile(t *testing.T) {
	ctx := context.Background()

	executor := &RealVNCServerTaskExecutor{
		FsManager: &utils.FsManager{},

		Reloader: &mockConfigReloader{},
	}

	newConfigFilename := "../realvnc/test/realvncserver-config-new.conf"
	defer os.Remove(newConfigFilename)

	tracker := &FieldNameStatusTracker{
		nameMap: withNameMap("idle_timeout", "IdleTimeout"),
		statusMap: fieldStatusMap{
			"IdleTimeout": FieldStatus{
				HasNewValue: true,
			},
		},
	}

	task := &RealVNCServerTask{
		Path:        "realvnc-server-1",
		ConfigFile:  newConfigFilename,
		IdleTimeout: 3600,
		SkipBackup:  false,
		mapper:      tracker,
		tracker:     tracker,
	}

	err := task.Validate(runtime.GOOS)
	require.NoError(t, err)

	res := executor.Execute(ctx, task)
	require.NoError(t, res.Err)
	require.True(t, task.Updated)

	assert.Equal(t, "Config updated", res.Comment)
	assert.Equal(t, "1 config value change(s) applied", res.Changes["count"])

	contents, err := os.ReadFile(task.ConfigFile)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "IdleTimeout=3600")

	info, err := os.Stat(task.ConfigFile)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(DefaultConfigFilePermissions), info.Mode().Perm())

	_, err = os.ReadFile(utils.GetBackupFilename(task.ConfigFile, "bak"))
	require.ErrorContains(t, err, "no such file")
}

func TestShouldRemoveSimpleConfigFileParam(t *testing.T) {
	testSetup(t)
	defer testTeardown(t)

	ctx := context.Background()

	executor := &RealVNCServerTaskExecutor{
		FsManager: &utils.FsManager{},

		Reloader: &mockConfigReloader{},
	}

	tracker := &FieldNameStatusTracker{
		nameMap: withNameMap("encryption", "Encryption"),
		statusMap: fieldStatusMap{
			"Encryption": FieldStatus{
				HasNewValue: true,
				Clear:       true,
			},
		},
	}

	task := &RealVNCServerTask{
		Path:       "realvnc-server-1",
		ConfigFile: "../realvnc/test/realvncserver-config.conf",
		Encryption: "!UNSET!",
		mapper:     tracker,
		tracker:    tracker,
	}

	err := task.Validate(runtime.GOOS)
	require.NoError(t, err)

	res := executor.Execute(ctx, task)
	require.NoError(t, res.Err)
	require.True(t, task.Updated)

	assert.Equal(t, "Config updated", res.Comment)
	assert.Equal(t, "1 config value change(s) applied", res.Changes["count"])

	contents, err := os.ReadFile(task.ConfigFile)
	require.NoError(t, err)

	assert.NotContains(t, string(contents), "Encryption")
}

func TestShouldMakeReloadCmdLine(t *testing.T) {
	cases := []struct {
		name            string
		task            RealVNCServerTask
		goos            string
		expectedCmdLine string
		expectedParams  []string
	}{
		{
			name: "linux default",
			task: RealVNCServerTask{
				Path:       "MyTask",
				ServerMode: UserServerMode,
			},
			goos:            "linux",
			expectedCmdLine: "/usr/bin/vncserver-x11",
			expectedParams:  []string{"-reload"},
		},
		{
			name: "linux user server mode",
			task: RealVNCServerTask{
				Path:       "MyTask",
				ServerMode: UserServerMode,
			},
			goos:            "linux",
			expectedCmdLine: "/usr/bin/vncserver-x11",
			expectedParams:  []string{"-reload"},
		},
		{
			name: "linux service server mode",
			task: RealVNCServerTask{
				Path:       "MyTask",
				ServerMode: ServiceServerMode,
			},
			goos:            "linux",
			expectedCmdLine: "/usr/bin/vncserver-x11",
			expectedParams:  []string{"-service", "-reload"},
		},
		{
			name: "linux virtual server mode",
			task: RealVNCServerTask{
				Path:       "MyTask",
				ServerMode: VirtualServerMode,
			},
			goos:            "linux",
			expectedCmdLine: "/usr/bin/vnclicense",
			expectedParams:  []string{"-reload"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			task := tc.task

			initMapperTracker(&task)

			err := task.Validate(tc.goos)
			require.NoError(t, err)

			cmdLine, params := makeReloadCmdLine(&task, tc.goos)
			assert.Equal(t, tc.expectedCmdLine, cmdLine)
			assert.Equal(t, tc.expectedParams, params)
		})
	}
}
