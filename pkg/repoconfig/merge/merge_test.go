package merge_test

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/repoconfig/merge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestMergeRepositoryConfig(t *testing.T) {
	testCases := []struct {
		name       string
		cfg        config.Config
		pluginCfg  plugins.Configuration
		repoConfig v1alpha1.RepositoryConfig
	}{
		{
			name: "emptyConfig",
			repoConfig: v1alpha1.RepositoryConfig{
				Spec: v1alpha1.RepositoryConfigSpec{
					Presubmits: []v1alpha1.Presubmit{
						{
							JobBase: v1alpha1.JobBase{
								Name: "lint",
							},
							AlwaysRun:    true,
							Optional:     false,
							Trigger:      "/lint",
							RerunCommand: "/relint",
							Reporter: v1alpha1.Reporter{
								Context: "lint",
							},
						},
					},
					Postsubmits: []v1alpha1.Postsubmit{
						{
							JobBase: v1alpha1.JobBase{
								Name: "release",
							},
							Reporter: v1alpha1.Reporter{
								Context: "release",
							},
						},
					},
					Keeper: &v1alpha1.Keeper{
						Query: &config.KeeperQuery{
							Labels:        []string{"mylabel"},
							MissingLabels: []string{"foo", "bar"},
						},
					},
				},
			},
		},
	}

	repoOwner := "myorg"
	repoName := "myowner"
	repoKey := repoOwner + "/" + repoName

	for _, tc := range testCases {
		name := tc.name
		err := merge.ConfigMerge(&tc.cfg, &tc.pluginCfg, &tc.repoConfig, repoOwner, repoName)
		require.NoError(t, err, "failed to merge repository config for %s", name)

		assert.Equal(t, len(tc.repoConfig.Spec.Presubmits), len(tc.cfg.Presubmits[repoKey]), "presubmits for %s", name)
		t.Logf("test %s has %d presubmits for repository key %s", name, len(tc.cfg.Presubmits[repoKey]), repoKey)

		assert.Equal(t, len(tc.repoConfig.Spec.Postsubmits), len(tc.cfg.Postsubmits[repoKey]), "postsubmits for %s", name)
		t.Logf("test %s has %d postsubmits for repository key %s", name, len(tc.cfg.Postsubmits[repoKey]), repoKey)
	}
}

func TestMergeRepositoryConfigFiles(t *testing.T) {
	sourceData := filepath.Join("test_data")
	fileNames, err := ioutil.ReadDir(sourceData)
	assert.NoError(t, err)

	repoOwner := "myorg"
	repoName := "myowner"

	for _, f := range fileNames {
		if f.IsDir() {
			name := f.Name()
			srcConfigFile := filepath.Join(sourceData, name, "source-config.yaml")
			srcPluginsFile := filepath.Join(sourceData, name, "source-plugins.yaml")
			expectedConfigFile := filepath.Join(sourceData, name, "expected-config.yaml")
			expectedPluginsFile := filepath.Join(sourceData, name, "expected-plugins.yaml")
			repoConfigFile := filepath.Join(sourceData, name, "lighthouse.yaml")
			require.FileExists(t, srcConfigFile)
			require.FileExists(t, expectedConfigFile)
			require.FileExists(t, expectedPluginsFile)
			require.FileExists(t, repoConfigFile)

			cfg := &config.Config{}
			pluginCfg := &plugins.Configuration{}
			repoConfig := &v1alpha1.RepositoryConfig{}
			LoadYAMLFile(t, srcConfigFile, cfg)
			LoadYAMLFile(t, srcPluginsFile, pluginCfg)
			LoadYAMLFile(t, repoConfigFile, repoConfig)

			err := merge.ConfigMerge(cfg, pluginCfg, repoConfig, repoOwner, repoName)
			require.NoError(t, err, "failed to merge files in dir %s", name)

			resultConfigText := ToYAMLString(t, cfg, name)
			resultPluginsText := ToYAMLString(t, pluginCfg, name)

			expectedConfigText := LoadTrimmedText(t, expectedConfigFile)
			expectedPluginsText := LoadTrimmedText(t, expectedPluginsFile)

			if d := cmp.Diff(strings.TrimSpace(resultConfigText), expectedConfigText); d != "" {
				t.Errorf("Generated config did not match expected: %s", d)
			}
			if d := cmp.Diff(strings.TrimSpace(resultPluginsText), expectedPluginsText); d != "" {
				t.Errorf("Generated plugins did not match expected: %s", d)
			}
			t.Logf("generated for file %s\n%s\n", srcConfigFile, resultConfigText)
			t.Logf("generated for file %s\n%s\n", srcPluginsFile, resultPluginsText)
		}
	}
}

func LoadTrimmedText(t *testing.T, expectedConfigFile string) string {
	expectData, err := ioutil.ReadFile(expectedConfigFile)
	require.NoError(t, err, "failed to load results %s", expectedConfigFile)
	expectedText := strings.TrimSpace(string(expectData))
	return expectedText
}

func ToYAMLString(t *testing.T, cfg interface{}, testName string) string {
	resultData, err := yaml.Marshal(&cfg)
	require.NoError(t, err, "failed to marshal config for %s", testName)
	resultText := string(resultData)
	return resultText
}

// LoadFile loads the given YAML file
func LoadYAMLFile(t *testing.T, fileName string, dest interface{}) {
	require.FileExists(t, fileName)

	data, err := ioutil.ReadFile(fileName)
	require.NoError(t, err, "failed to read file %s", fileName)

	err = yaml.Unmarshal(data, dest)
	require.NoError(t, err, "failed to unmarshal YAML file %s", fileName)
}
