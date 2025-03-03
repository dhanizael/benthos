package test_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Jeffail/benthos/v3/lib/message"
	"github.com/Jeffail/benthos/v3/lib/processor"
	"github.com/Jeffail/benthos/v3/lib/service/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"

	_ "github.com/Jeffail/benthos/v3/public/components/all"
)

func initTestFiles(files map[string]string) (string, error) {
	testDir, err := ioutil.TempDir("", "benthos_config_test_test")
	if err != nil {
		return "", err
	}

	for k, v := range files {
		fp := filepath.Join(testDir, k)
		if err := os.MkdirAll(filepath.Dir(fp), 0777); err != nil {
			return "", err
		}
		if err := ioutil.WriteFile(fp, []byte(v), 0777); err != nil {
			return "", err
		}
	}

	return testDir, nil
}

func TestProcessorsProviderErrors(t *testing.T) {
	files := map[string]string{
		"config1.yaml": `
this isnt valid yaml
		nah
		what is even happening here?`,
		"config2.yaml": `
pipeline:
  processors:
  - type: text`,
		"config3.yaml": `
pipeline:
  processors:
  - type: doesnotexist`,
	}

	testDir, err := initTestFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	if _, err = test.NewProcessorsProvider(filepath.Join(testDir, "doesnotexist.yaml")).Provide("/pipeline/processors", nil); err == nil {
		t.Error("Expected error from bad filepath")
	}
	if _, err = test.NewProcessorsProvider(filepath.Join(testDir, "config1.yaml")).Provide("/pipeline/processors", nil); err == nil {
		t.Error("Expected error from bad config file")
	}
	if _, err = test.NewProcessorsProvider(filepath.Join(testDir, "config2.yaml")).Provide("/not/a/valid/path", nil); err == nil {
		t.Error("Expected error from bad processors path")
	}
	if _, err = test.NewProcessorsProvider(filepath.Join(testDir, "config3.yaml")).Provide("/pipeline/processors", nil); err == nil {
		t.Error("Expected error from bad processor type")
	}
}

func TestProcessorsProviderDeprecated(t *testing.T) {
	files := map[string]string{
		"config1.yaml": `
resources:
  caches:
    foocache:
      memory: {}

pipeline:
  processors:
  - metadata:
      operator: set
      key: foo
      value: ${FOO_VAR:defaultvalue}
  - cache:
      cache: foocache
      operator: set
      key: defaultkey
      value: ${! meta("foo") }
  - cache:
      cache: foocache
      operator: get
      key: defaultkey
  - text:
      operator: to_upper`,

		"config2.yaml": `
resources:
  caches:
    foocache:
      memory: {}

pipeline:
  processors:
    $ref: ./config1.yaml#/pipeline/processors`,
	}

	testDir, err := initTestFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	provider := test.NewProcessorsProvider(filepath.Join(testDir, "config1.yaml"))
	procs, err := provider.Provide("/pipeline/processors", nil)
	if err != nil {
		t.Fatal(err)
	}
	if exp, act := 4, len(procs); exp != act {
		t.Fatalf("Unexpected processor count: %v != %v", act, exp)
	}
	msgs, res := processor.ExecuteAll(procs, message.New([][]byte{[]byte("hello world")}))
	if res != nil {
		t.Fatal(res.Error())
	}
	if exp, act := "DEFAULTVALUE", string(msgs[0].Get(0).Get()); exp != act {
		t.Errorf("Unexpected result: %v != %v", act, exp)
	}

	if procs, err = provider.Provide("/pipeline/processors", map[string]string{
		"FOO_VAR": "newvalue",
	}); err != nil {
		t.Fatal(err)
	}
	if exp, act := 4, len(procs); exp != act {
		t.Fatalf("Unexpected processor count: %v != %v", act, exp)
	}
	if msgs, res = processor.ExecuteAll(procs, message.New([][]byte{[]byte("hello world")})); res != nil {
		t.Fatal(res.Error())
	}
	if exp, act := "NEWVALUE", string(msgs[0].Get(0).Get()); exp != act {
		t.Errorf("Unexpected result: %v != %v", act, exp)
	}

	provider = test.NewProcessorsProvider(filepath.Join(testDir, "config2.yaml"))
	if procs, err = provider.Provide("/pipeline/processors", map[string]string{
		"FOO_VAR": "thirdvalue",
	}); err != nil {
		t.Fatal(err)
	}
	if exp, act := 4, len(procs); exp != act {
		t.Fatalf("Unexpected processor count: %v != %v", act, exp)
	}
	if msgs, res = processor.ExecuteAll(procs, message.New([][]byte{[]byte("hello world")})); res != nil {
		t.Fatal(res.Error())
	}
	if exp, act := "THIRDVALUE", string(msgs[0].Get(0).Get()); exp != act {
		t.Errorf("Unexpected result: %v != %v", act, exp)
	}

	if procs, err = provider.Provide("/pipeline/processors/3", nil); err != nil {
		t.Fatal(err)
	}
	if exp, act := 1, len(procs); exp != act {
		t.Fatalf("Unexpected processor count: %v != %v", act, exp)
	}
	if msgs, res = processor.ExecuteAll(procs, message.New([][]byte{[]byte("hello world")})); res != nil {
		t.Fatal(res.Error())
	}
	if exp, act := "HELLO WORLD", string(msgs[0].Get(0).Get()); exp != act {
		t.Errorf("Unexpected result: %v != %v", act, exp)
	}
}

func TestProcessorsExtraResourcesDeprecated(t *testing.T) {
	files := map[string]string{
		"resources1.yaml": `
resources:
  caches:
    barcache:
      memory: {}
`,
		"resources2.yaml": `
resources:
  caches:
    bazcache:
      memory: {}
`,
		"config1.yaml": `
resources:
  caches:
    foocache:
      memory: {}

pipeline:
  processors:
  - cache:
      cache: foocache
      operator: set
      key: defaultkey
      value: foo
  - cache:
      cache: barcache
      operator: set
      key: defaultkey
      value: bar
  - cache:
      cache: bazcache
      operator: set
      key: defaultkey
      value: bar
`,
	}

	testDir, err := initTestFiles(files)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	provider := test.NewProcessorsProvider(
		filepath.Join(testDir, "config1.yaml"),
		test.OptAddResourcesPaths([]string{
			filepath.Join(testDir, "resources1.yaml"),
			filepath.Join(testDir, "resources2.yaml"),
		}),
	)
	procs, err := provider.Provide("/pipeline/processors", nil)
	require.NoError(t, err)
	assert.Len(t, procs, 3)
}

func TestProcessorsExtraResourcesErrorDeprecated(t *testing.T) {
	files := map[string]string{
		"resources1.yaml": `
resources:
  caches:
    barcache:
      memory: {}
`,
		"resources2.yaml": `
resources:
  caches:
    barcache:
      memory: {}
`,
		"config1.yaml": `
resources:
  caches:
    foocache:
      memory: {}

pipeline:
  processors:
  - cache:
      cache: foocache
      operator: set
      key: defaultkey
      value: foo
  - cache:
      cache: barcache
      operator: set
      key: defaultkey
      value: bar
`,
	}

	testDir, err := initTestFiles(files)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	provider := test.NewProcessorsProvider(
		filepath.Join(testDir, "config1.yaml"),
		test.OptAddResourcesPaths([]string{
			filepath.Join(testDir, "resources1.yaml"),
			filepath.Join(testDir, "resources2.yaml"),
		}),
	)
	_, err = provider.Provide("/pipeline/processors", nil)
	require.EqualError(t, err, fmt.Sprintf("failed to merge resources from '%v/resources2.yaml': resource cache name collision: barcache", testDir))
}

func TestProcessorsProvider(t *testing.T) {
	files := map[string]string{
		"config1.yaml": `
cache_resources:
  - label: foocache
    memory: {}

pipeline:
  processors:
  - metadata:
      operator: set
      key: foo
      value: ${BAR_VAR:defaultvalue}
  - cache:
      cache: foocache
      operator: set
      key: defaultkey
      value: ${! meta("foo") }
  - cache:
      cache: foocache
      operator: get
      key: defaultkey
  - text:
      operator: to_upper`,

		"config2.yaml": `
cache_resources:
  - label: foocache
    memory: {}

pipeline:
  processors:
    $ref: ./config1.yaml#/pipeline/processors`,
	}

	testDir, err := initTestFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir)

	provider := test.NewProcessorsProvider(filepath.Join(testDir, "config1.yaml"))
	procs, err := provider.Provide("/pipeline/processors", nil)
	if err != nil {
		t.Fatal(err)
	}
	if exp, act := 4, len(procs); exp != act {
		t.Fatalf("Unexpected processor count: %v != %v", act, exp)
	}
	msgs, res := processor.ExecuteAll(procs, message.New([][]byte{[]byte("hello world")}))
	if res != nil {
		t.Fatal(res.Error())
	}
	if exp, act := "DEFAULTVALUE", string(msgs[0].Get(0).Get()); exp != act {
		t.Errorf("Unexpected result: %v != %v", act, exp)
	}

	if procs, err = provider.Provide("/pipeline/processors", map[string]string{
		"BAR_VAR": "newvalue",
	}); err != nil {
		t.Fatal(err)
	}
	if exp, act := 4, len(procs); exp != act {
		t.Fatalf("Unexpected processor count: %v != %v", act, exp)
	}
	if msgs, res = processor.ExecuteAll(procs, message.New([][]byte{[]byte("hello world")})); res != nil {
		t.Fatal(res.Error())
	}
	if exp, act := "NEWVALUE", string(msgs[0].Get(0).Get()); exp != act {
		t.Errorf("Unexpected result: %v != %v", act, exp)
	}

	provider = test.NewProcessorsProvider(filepath.Join(testDir, "config2.yaml"))
	if procs, err = provider.Provide("/pipeline/processors", map[string]string{
		"BAR_VAR": "thirdvalue",
	}); err != nil {
		t.Fatal(err)
	}
	if exp, act := 4, len(procs); exp != act {
		t.Fatalf("Unexpected processor count: %v != %v", act, exp)
	}
	if msgs, res = processor.ExecuteAll(procs, message.New([][]byte{[]byte("hello world")})); res != nil {
		t.Fatal(res.Error())
	}
	if exp, act := "THIRDVALUE", string(msgs[0].Get(0).Get()); exp != act {
		t.Errorf("Unexpected result: %v != %v", act, exp)
	}

	if procs, err = provider.Provide("/pipeline/processors/3", nil); err != nil {
		t.Fatal(err)
	}
	if exp, act := 1, len(procs); exp != act {
		t.Fatalf("Unexpected processor count: %v != %v", act, exp)
	}
	if msgs, res = processor.ExecuteAll(procs, message.New([][]byte{[]byte("hello world")})); res != nil {
		t.Fatal(res.Error())
	}
	if exp, act := "HELLO WORLD", string(msgs[0].Get(0).Get()); exp != act {
		t.Errorf("Unexpected result: %v != %v", act, exp)
	}
}

func TestProcessorsProviderMocks(t *testing.T) {
	files := map[string]string{
		"config1.yaml": `
pipeline:
  processors:
    - http:
        url: http://example.com/foobar
        verb: POST
    - bloblang: 'root = content().string() + " first proc"'
    - http:
        url: http://example.com/barbaz
        verb: POST
    - bloblang: 'root = content().string() + " second proc"'
`,
	}

	testDir, err := initTestFiles(files)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(testDir)
	})

	mocks := map[string]yaml.Node{}
	require.NoError(t, yaml.Unmarshal([]byte(`
"/pipeline/processors/0":
  bloblang: 'root = content().string() + " first mock"'
"/pipeline/processors/2":
  bloblang: 'root = content().string() + " second mock"'
`), &mocks))

	provider := test.NewProcessorsProvider(filepath.Join(testDir, "config1.yaml"))
	procs, err := provider.ProvideMocked("/pipeline/processors", nil, mocks)
	require.NoError(t, err)

	require.Len(t, procs, 4)

	msgs, res := processor.ExecuteAll(procs, message.New([][]byte{[]byte("starts with")}))
	require.Nil(t, res)
	require.Len(t, msgs, 1)
	require.Equal(t, 1, msgs[0].Len())

	assert.Equal(t, "starts with first mock first proc second mock second proc", string(msgs[0].Get(0).Get()))
}

func TestProcessorsProviderMocksFromLabel(t *testing.T) {
	files := map[string]string{
		"config1.yaml": `
pipeline:
  processors:
    - label: first_http
      http:
        url: http://example.com/foobar
        verb: POST
    - bloblang: 'root = content().string() + " first proc"'
    - label: second_http
      http:
        url: http://example.com/barbaz
        verb: POST
    - bloblang: 'root = content().string() + " second proc"'
`,
	}

	testDir, err := initTestFiles(files)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(testDir)
	})

	mocks := map[string]yaml.Node{}
	require.NoError(t, yaml.Unmarshal([]byte(`
"first_http":
  bloblang: 'root = content().string() + " first mock"'
"second_http":
  bloblang: 'root = content().string() + " second mock"'
`), &mocks))

	provider := test.NewProcessorsProvider(filepath.Join(testDir, "config1.yaml"))
	procs, err := provider.ProvideMocked("/pipeline/processors", nil, mocks)
	require.NoError(t, err)

	require.Len(t, procs, 4)

	msgs, res := processor.ExecuteAll(procs, message.New([][]byte{[]byte("starts with")}))
	require.Nil(t, res)
	require.Len(t, msgs, 1)
	require.Equal(t, 1, msgs[0].Len())

	assert.Equal(t, "starts with first mock first proc second mock second proc", string(msgs[0].Get(0).Get()))
}

func TestProcessorsProviderMocksMixed(t *testing.T) {
	files := map[string]string{
		"config1.yaml": `
pipeline:
  processors:
    - label: first_http
      http:
        url: http://example.com/foobar
        verb: POST
    - bloblang: 'root = content().string() + " first proc"'
    - label: second_http
      http:
        url: http://example.com/barbaz
        verb: POST
    - bloblang: 'root = content().string() + " second proc"'
`,
	}

	testDir, err := initTestFiles(files)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(testDir)
	})

	mocks := map[string]yaml.Node{}
	require.NoError(t, yaml.Unmarshal([]byte(`
"first_http":
  bloblang: 'root = content().string() + " first mock"'
"/pipeline/processors/2":
  bloblang: 'root = content().string() + " second mock"'
`), &mocks))

	provider := test.NewProcessorsProvider(filepath.Join(testDir, "config1.yaml"))
	procs, err := provider.ProvideMocked("/pipeline/processors", nil, mocks)
	require.NoError(t, err)

	require.Len(t, procs, 4)

	msgs, res := processor.ExecuteAll(procs, message.New([][]byte{[]byte("starts with")}))
	require.Nil(t, res)
	require.Len(t, msgs, 1)
	require.Equal(t, 1, msgs[0].Len())

	assert.Equal(t, "starts with first mock first proc second mock second proc", string(msgs[0].Get(0).Get()))
}

func TestProcessorsExtraResources(t *testing.T) {
	files := map[string]string{
		"resources1.yaml": `
cache_resources:
  - label: barcache
    memory: {}
`,
		"resources2.yaml": `
cache_resources:
  - label: bazcache
    memory: {}
`,
		"config1.yaml": `
cache_resources:
  - label: foocache
    memory: {}

pipeline:
  processors:
  - cache:
      cache: foocache
      operator: set
      key: defaultkey
      value: foo
  - cache:
      cache: barcache
      operator: set
      key: defaultkey
      value: bar
  - cache:
      cache: bazcache
      operator: set
      key: defaultkey
      value: bar
`,
	}

	testDir, err := initTestFiles(files)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	provider := test.NewProcessorsProvider(
		filepath.Join(testDir, "config1.yaml"),
		test.OptAddResourcesPaths([]string{
			filepath.Join(testDir, "resources1.yaml"),
			filepath.Join(testDir, "resources2.yaml"),
		}),
	)
	procs, err := provider.Provide("/pipeline/processors", nil)
	require.NoError(t, err)
	assert.Len(t, procs, 3)
}

func TestProcessorsExtraResourcesError(t *testing.T) {
	files := map[string]string{
		"resources1.yaml": `
cache_resources:
  - label: barcache
    memory: {}
`,
		"resources2.yaml": `
cache_resources:
  - label: barcache
    memory: {}
`,
		"config1.yaml": `
cache_resources:
  - label: foocache
    memory: {}

pipeline:
  processors:
  - cache:
      cache: foocache
      operator: set
      key: defaultkey
      value: foo
  - cache:
      cache: barcache
      operator: set
      key: defaultkey
      value: bar
`,
	}

	testDir, err := initTestFiles(files)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	provider := test.NewProcessorsProvider(
		filepath.Join(testDir, "config1.yaml"),
		test.OptAddResourcesPaths([]string{
			filepath.Join(testDir, "resources1.yaml"),
			filepath.Join(testDir, "resources2.yaml"),
		}),
	)
	_, err = provider.Provide("/pipeline/processors", nil)
	require.EqualError(t, err, "failed to initialise resources: cache resource label 'barcache' collides with a previously defined resource")
}
