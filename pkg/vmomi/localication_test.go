package vmomi

import "testing"

//revive:disable:add-constant

func Test_parseCatalog_EmptyContent(t *testing.T) {
	content := ""
	catalog := parseCatalog(&content)

	if len(catalog) != 0 {
		t.Errorf("Invalid catalog: %v", catalog)
	}
}

func Test_parseCatalog_ZeroKV(t *testing.T) {
	content := "key:value"
	catalog := parseCatalog(&content)

	if len(catalog) != 0 {
		t.Errorf("Invalid catalog: %v", catalog)
	}
}

func Test_parseCatalog_OneKV(t *testing.T) {
	content := "key=value"
	catalog := parseCatalog(&content)

	v, ok := catalog["key"]
	if v != "value" || !ok {
		t.Errorf("Invalid catalog: %v", catalog)
	}
}

func Test_parseCatalog_TwoKV(t *testing.T) {
	content := `key1=value1
key2=value2
`
	catalog := parseCatalog(&content)

	v, ok := catalog["key1"]
	if v != "value1" || !ok {
		t.Errorf("Invalid catalog: %v", catalog)
	}

	v, ok = catalog["key2"]
	if v != "value2" || !ok {
		t.Errorf("Invalid catalog: %v", catalog)
	}
}

func Test_parseCatalog_MultilineKV(t *testing.T) {
	content := `key1=value1-a\
value1-b\
value1-c
key2=value2
`
	catalog := parseCatalog(&content)

	v, ok := catalog["key1"]
	if v != "value1-avalue1-bvalue1-c" || !ok {
		t.Errorf("Invalid catalog: %v", catalog)
	}

	v, ok = catalog["key2"]
	if v != "value2" || !ok {
		t.Errorf("Invalid catalog: %v", catalog)
	}
}

func Test_parseCatalog_IgnoreComment(t *testing.T) {
	content := `# comment
key="value"
`
	catalog := parseCatalog(&content)

	v, ok := catalog["key"]
	if v != "value" || !ok {
		t.Errorf("Invalid catalog: %v", catalog)
	}
}

//revive:enable:add-constant
