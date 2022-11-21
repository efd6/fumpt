package main

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/google/go-cmp/cmp"
)

var canonicalQuotesTests = []struct {
	name string
	in   string
	want string
	skip string
}{
	{
		name: "single_quoted",
		in:   `key: 'string'`,
		want: `key: string`,
	},
	{
		name: "double_quoted",
		in:   `key: "string"`,
		want: `key: string`,
	},
	{
		name: "single_quoted_empty",
		in:   `key: ''`,
		want: `key: ''`,
	},
	{
		name: "double_quoted_empty",
		in:   `key: ""`,
		want: `key: ''`,
	},
	{
		name: "single_quoted_space",
		in:   `key: 'string with space'`,
		want: `key: string with space`,
	},
	{
		name: "double_quoted_space",
		in:   `key: "string with space"`,
		want: `key: string with space`,
	},
	{
		name: "single_quoted_timestamp",
		in:   `key: '@timestamp'`,
		want: `key: '@timestamp'`,
	},
	{
		name: "double_quoted_timestamp",
		in:   `key: "@timestamp"`,
		want: `key: '@timestamp'`,
	},
	{
		name: "single_quoted_semver",
		in:   `key: '1.2.3'`,
		want: `key: 1.2.3`,
	},
	{
		name: "double_quoted_semver",
		in:   `key: "1.2.3"`,
		want: `key: 1.2.3`,
	},
	{
		name: "single_quoted_semver_hat",
		in:   `key: '^1.2.0'`,
		want: `key: ^1.2.0`,
	},
	{
		name: "double_quoted_semver_hat",
		in:   `key: "^1.2.0"`,
		want: `key: ^1.2.0`,
	},
	{
		name: "single_quoted_semver_alternation",
		in:   `key: '1.2.3 || 2.0.0'`,
		want: `key: 1.2.3 || 2.0.0`,
	},
	{
		name: "double_quoted_semver_alternation",
		in:   `key: "1.2.3 || 2.0.0"`,
		want: `key: 1.2.3 || 2.0.0`,
	},
	{
		name: "single_quoted_semver_no_patch",
		in:   `key: '1.2'`,
		want: `key: '1.2'`,
	},
	{
		name: "double_quoted_semver_no_patch",
		in:   `key: "1.2"`,
		want: `key: '1.2'`,
	},
	{
		name: "single_quoted_new_line",
		in:   `key: 'one\ntwo'`,
		want: `key: one\ntwo`,
	},
	{
		name: "double_quoted_new_line",
		in:   `key: "one\ntwo"`,
		want: `key: "one\ntwo"`,
	},
	{
		name: "new_line_bar",
		in: `key: |
  one
  two'`,
		want: `key: |
  one
  two'`,
	},
	{
		name: "new_line_arrow",
		in: `key: >
  one
  two'`,
		want: `key: >
  one
  two'`,
	},
	{
		name: "if_not_single_quoted",
		in:   `if: '!ok'`,
		want: `if: '!ok'`,
	},
	{
		name: "if_not_double_quoted",
		in:   `if: "!ok"`,
		want: `if: '!ok'`,
	},
	{
		name: "if_not_single_quoted_with_double_quoted_string",
		in:   `if: '!ok("string")'`,
		want: `if: '!ok("string")'`,
	},
	{
		name: "if_not_double_quoted_with_single_quoted_string",
		in:   `if: "!ok('string')"`,
		want: `if: "!ok('string')"`,
	},
	{
		name: "if_single_quoted_with_double_quoted_string",
		in:   `if: 'ok("string")'`,
		want: `if: ok("string")`,
	},
	{
		name: "if_double_quoted_with_single_quoted_string",
		in:   `if: "ok('string')"`,
		want: `if: ok('string')`,
	},
	{
		name: "single_quote_spaced_keys",
		in:   `'key with spaces': 'value'`,
		want: `'key with spaces': 'value'`,
	},
	{
		name: "double_quote_spaced_keys",
		in:   `"key with spaces": 'value'`,
		want: `'key with spaces': 'value'`,
	},
	{
		name: "number_key_noquote",
		in: `
map:
  "123": "string1"
  "456": "string2"
`,
		want: `map:
  123: string1
  456: string2`,
	},
	{
		name: "quote_key_single_quote_val",
		in: `map:
  "key1": 'string1'
  "key2": 'string2'
`,
		want: `map:
  key1: string1
  key2: string2`,
	},
	{
		name: "quote_key_double_quote_val",
		in: `map:
  "key1": "string1"
  "key2": "string2"
`,
		want: `map:
  key1: string1
  key2: string2`,
	},
	{
		name: "quote_key_spaces_single_quote_val",
		in: `map:
  "key 1": 'string1'
  "key 2": 'string2'
`,
		want: `map:
  'key 1': 'string1'
  'key 2': 'string2'`,
	},
	{
		name: "quote_key_spaces_double_quote_val",
		in: `map:
  "key 1": "string1"
  "key 2": "string2"
`,
		want: `map:
  'key 1': 'string1'
  'key 2': 'string2'`,
	},
	{
		name: "quote_key_spaces_double_quote_escaped_val",
		in: `map:
  "key 1": "string\n1"
  "key 2": "string\n2"
`,
		want: `map:
  'key 1': "string\n1"
  'key 2': "string\n2"`,
	},
	{
		skip: skipReasonGoYAML323,
		name: "quote_key_noquote_val",
		in: `map:
  "key1": string1
  "key2": string2
`,
		want: `map:
  key1: string1
  key2: string2`,
	},
}

func TestCanonicalQuotes(t *testing.T) {
	for _, test := range canonicalQuotesTests {
		t.Run(test.name, func(t *testing.T) {
			if test.skip != "" {
				t.Skipf("skipping %s: %s", test.name, test.skip)
			}
			file, err := parser.ParseBytes([]byte(test.in), parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse document: %v", err)
			}
			for _, doc := range file.Docs {
				ast.Walk(canonicalQuotes{root: doc}, doc)
			}
			got := file.String()

			if got != test.want {
				t.Errorf("unexpected result:\n--- got\n+++ want\n%s", cmp.Diff(got, test.want))
			}
		})
	}
}

var canonicalOrderTests = []struct {
	name  string
	in    string
	order canonicalOrder
	want  string
}{
	{
		name: "manifest",
		in:   manifest,
		order: canonicalOrder{
			"$.name":        0,
			"$.title":       1,
			"$.version":     2,
			"$.description": 3,
			"$.owner":       -1, // owner last
		},
		want: `name: package
title: Integration package
version: 1.2.3
description: Do things with inputs
format_version: 1.0.0
policy_templates:
  # TODO: use new input
  - description: Collect logs
    inputs:
      - description: Collecting logs
        title: Collect logs from inputs
        type: logfile # filestream?
    name: logpkg
    title: Input logs
owner:
  github: elastic/security-external-integrations`,
	},
	{
		name: "manifest_relaxed_spec",
		in:   manifest,
		order: canonicalOrder{
			"*.name":        0,
			"*.title":       1,
			"$.version":     2,
			"*.description": 3,
			"$.owner":       -1,
		},
		want: `name: package
title: Integration package
version: 1.2.3
description: Do things with inputs
format_version: 1.0.0
policy_templates:
  # TODO: use new input
  - name: logpkg
    title: Input logs
    description: Collect logs
    inputs:
      - title: Collect logs from inputs
        description: Collecting logs
        type: logfile # filestream?
owner:
  github: elastic/security-external-integrations`,
	},
	{
		name: "changelog_strict_spec",
		in:   changelog,
		order: canonicalOrder{
			"$[*].version":                0,
			"$[*].changes":                1,
			"$[*].changes[*].description": 0, // *.description
			"$[*].changes[*].type":        1, // *.type
			"$[*].changes[*].link":        2, // *.link
		},
		want: `# newest first
- version: 1.2.3
  changes:
    - description: Add feature
      type: enhancement
      link: magic link 2
    - description: Fix bug
      type: bugfix
      link: magic link 3
- version: 0.1.0
  changes:
    - description: New package
      type: enhancement
      link: magic link 1`,
	},
	{
		name: "changelog_relaxed_spec",
		in:   changelog,
		order: canonicalOrder{
			"*.version":     0,
			"*.changes":     1,
			"*.description": 0,
			"*.type":        1,
			"*.link":        2,
		},
		want: `# newest first
- version: 1.2.3
  changes:
    - description: Add feature
      type: enhancement
      link: magic link 2
    - description: Fix bug
      type: bugfix
      link: magic link 3
- version: 0.1.0
  changes:
    - description: New package
      type: enhancement
      link: magic link 1`,
	},
	{
		name: "pipeling",
		in: `---
description: Pipeline for package logs
processors:
  - set:
      field: ecs.version
      value: '8.5.0'
  - rename:
      field: message
      target_field: event.original

  - set:
      tags: ["ordering"]
      field: _conf.tz_offset
      value: UTC
      override: false
  - set:
      field: event.timezone
      copy_from: _conf.tz_offset
      if: ctx.event?.timezone == null || ctx.event?.timezone == ""

  - remove:
      field:
        - _tmp
        - _conf
      ignore_missing: true

on_failure:
  - remove:
      field:
        - _tmp
        - _conf
      ignore_missing: true
  - append:
      field: error.message
      value: "{{ _ingest.on_failure_message }}"
`,
		order: canonicalOrder{
			"*.description": 0,
			"*.if":          1,
			"*.field":       2,
			"*.override":    -3,
			"*.tags":        -2,
			"*.on_failure":  -1,
		},
		want: `---
description: Pipeline for package logs
processors:
  - set:
      field: ecs.version
      value: '8.5.0'
  - rename:
      field: message
      target_field: event.original
  - set:
      field: _conf.tz_offset
      value: UTC
      override: false
      tags: ["ordering"]
  - set:
      if: ctx.event?.timezone == null || ctx.event?.timezone == ""
      field: event.timezone
      copy_from: _conf.tz_offset
  - remove:
      field:
        - _tmp
        - _conf
      ignore_missing: true
on_failure:
  - remove:
      field:
        - _tmp
        - _conf
      ignore_missing: true
  - append:
      field: error.message
      value: "{{ _ingest.on_failure_message }}"`,
	},
}

func TestCanonicalOrder(t *testing.T) {
	for _, test := range canonicalOrderTests {
		t.Run(test.name, func(t *testing.T) {
			file, err := parser.ParseBytes([]byte(test.in), parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse document: %v", err)
			}
			for _, doc := range file.Docs {
				ast.Walk(test.order, doc)
			}
			got := strings.TrimSpace(file.String())

			if got != test.want {
				t.Errorf("unexpected result:\n--- got\n+++ want\n%s", cmp.Diff(got, test.want))
			}
		})
	}
}

var sortListsTests = []struct {
	name  string
	in    string
	order sortLists
	want  string
}{
	{
		name: "fields",
		in:   fields,
		order: sortLists{
			canSort: isECSgroup,
			less:    lessByName,
		},
		want: `- name: cloud
  title: Cloud
  group: 2
  description: Fields related to the cloud or infrastructure the events are coming from.
  footnote: 'Examples: If Metricbeat is running on an EC2 host and fetches data from its host, the cloud info contains the data about this machine. If Metricbeat runs on a remote machine outside the cloud and fetches data from a service running in the cloud, the field contains cloud data from the machine the service is running on.'
  type: group
  fields:
    - name: account.id
      level: extended
      type: keyword
      ignore_above: 1024
      description: 'The cloud account or organization id used to identify different entities in a multi-tenant environment.  Examples: AWS account id, Google Cloud ORG Id, or other unique identifier.'
      example: 666777888999
    - name: availability_zone
      level: extended
      type: keyword
      ignore_above: 1024
      description: Availability zone in which this host is running.
      example: us-east-1c
    - name: instance.id
      level: extended
      type: keyword
      ignore_above: 1024
      description: Instance ID of the host machine.
      example: i-1234567890abcdef0`,
	},
	{
		name: "fields_one_missing_name",
		in:   fieldsOneMissingName,
		order: sortLists{
			canSort: isECSgroup,
			less:    lessByName,
		},
		want: `- name: cloud
  title: Cloud
  group: 2
  description: Fields related to the cloud or infrastructure the events are coming from.
  footnote: 'Examples: If Metricbeat is running on an EC2 host and fetches data from its host, the cloud info contains the data about this machine. If Metricbeat runs on a remote machine outside the cloud and fetches data from a service running in the cloud, the field contains cloud data from the machine the service is running on.'
  type: group
  fields:
    - name: account.id
      level: extended
      type: keyword
      ignore_above: 1024
      description: 'The cloud account or organization id used to identify different entities in a multi-tenant environment.  Examples: AWS account id, Google Cloud ORG Id, or other unique identifier.'
      example: 666777888999
    - name: instance.id
      level: extended
      type: keyword
      ignore_above: 1024
      description: Instance ID of the host machine.
      example: i-1234567890abcdef0
    - level: extended
      type: keyword
      ignore_above: 1024
      description: Availability zone in which this host is running.
      example: us-east-1c`,
	},
	{
		name: "fields_two_missing_names",
		in:   fieldsTwoMissingNames,
		order: sortLists{
			canSort: isECSgroup,
			less:    lessByName,
		},
		want: `- name: cloud
  title: Cloud
  group: 2
  description: Fields related to the cloud or infrastructure the events are coming from.
  footnote: 'Examples: If Metricbeat is running on an EC2 host and fetches data from its host, the cloud info contains the data about this machine. If Metricbeat runs on a remote machine outside the cloud and fetches data from a service running in the cloud, the field contains cloud data from the machine the service is running on.'
  type: group
  fields:
    - name: instance.id
      level: extended
      type: keyword
      ignore_above: 1024
      description: Instance ID of the host machine.
      example: i-1234567890abcdef0
    - level: extended
      type: keyword
      ignore_above: 1024
      description: Availability zone in which this host is running.
      example: us-east-1c
    - level: extended
      type: keyword
      ignore_above: 1024
      description: 'The cloud account or organization id used to identify different entities in a multi-tenant environment.  Examples: AWS account id, Google Cloud ORG Id, or other unique identifier.'
      example: 666777888999`,
	},
	{
		name: "ecs_fields",
		in: `- external: ecs
  name: error.message
- external: ecs
  name: event.category
- external: ecs
  name: "@timestamp"
- external: ecs
  name: event.code
- external: ecs
  name: event.created
`,
		order: sortLists{
			canSort: isECSgroup,
			less:    lessByName,
		},
		want: `- external: ecs
  name: "@timestamp"
- external: ecs
  name: error.message
- external: ecs
  name: event.category
- external: ecs
  name: event.code
- external: ecs
  name: event.created`,
	},
	{
		name: "ecs_fields_no_unnamed",
		in: `fields:
  - external: ecs
    name: error.message
  - external: ecs
    name: event.category
  - external: ecs
    name: "@timestamp"
  - external: ecs
    name: event.code
  - external: ecs
    name: event.created
`,
		order: sortLists{
			canSort: isECSgroup,
			less:    lessByName,
		},
		want: `fields:
  - external: ecs
    name: error.message
  - external: ecs
    name: event.category
  - external: ecs
    name: "@timestamp"
  - external: ecs
    name: event.code
  - external: ecs
    name: event.created`,
	},
	{
		name: "ecs_fields_no_named",
		in: `- name: fail
  fields:
    - external: ecs
      name: error.message
    - external: ecs
      name: event.category
    - external: ecs
      name: "@timestamp"
    - external: ecs
      name: event.code
    - external: ecs
      name: event.created
`,
		order: sortLists{
			canSort: isECSgroup,
			less:    lessByName,
		},
		want: `- name: fail
  fields:
    - external: ecs
      name: error.message
    - external: ecs
      name: event.category
    - external: ecs
      name: "@timestamp"
    - external: ecs
      name: event.code
    - external: ecs
      name: event.created`,
	},
}

func TestSortLists(t *testing.T) {
	for _, test := range sortListsTests {
		t.Run(test.name, func(t *testing.T) {
			file, err := parser.ParseBytes([]byte(test.in), parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse document: %v", err)
			}
			for _, doc := range file.Docs {
				test.order.root = doc
				ast.Walk(test.order, doc)
			}
			got := strings.TrimSpace(file.String())

			if got != test.want {
				t.Errorf("unexpected result:\n--- got\n+++ want\n%s", cmp.Diff(got, test.want))
			}
		})
	}
}

const (
	manifest = `format_version: 1.0.0
name: package
title: Integration package
version: 1.2.3
description: Do things with inputs
policy_templates:
  # TODO: use new input
  - name: logpkg
    title: Input logs
    description: Collect logs
    inputs:
      - description: Collecting logs
        title: Collect logs from inputs
        type: logfile # filestream?
owner:
  github: elastic/security-external-integrations
`

	changelog = `# newest first
- version: 1.2.3
  changes:
    - description: Add feature
      link: magic link 2
      type: enhancement
    - description: Fix bug
      link: magic link 3
      type: bugfix
- version: 0.1.0
  changes:
    - description: New package
      type: enhancement
      link: magic link 1
`

	fields = `- name: cloud
  title: Cloud
  group: 2
  description: Fields related to the cloud or infrastructure the events are coming from.
  footnote: 'Examples: If Metricbeat is running on an EC2 host and fetches data from its host, the cloud info contains the data about this machine. If Metricbeat runs on a remote machine outside the cloud and fetches data from a service running in the cloud, the field contains cloud data from the machine the service is running on.'
  type: group
  fields:
    - name: instance.id
      level: extended
      type: keyword
      ignore_above: 1024
      description: Instance ID of the host machine.
      example: i-1234567890abcdef0
    - name: availability_zone
      level: extended
      type: keyword
      ignore_above: 1024
      description: Availability zone in which this host is running.
      example: us-east-1c
    - name: account.id
      level: extended
      type: keyword
      ignore_above: 1024
      description: 'The cloud account or organization id used to identify different entities in a multi-tenant environment.

        Examples: AWS account id, Google Cloud ORG Id, or other unique identifier.'
      example: 666777888999
`

	fieldsOneMissingName = `- name: cloud
  title: Cloud
  group: 2
  description: Fields related to the cloud or infrastructure the events are coming from.
  footnote: 'Examples: If Metricbeat is running on an EC2 host and fetches data from its host, the cloud info contains the data about this machine. If Metricbeat runs on a remote machine outside the cloud and fetches data from a service running in the cloud, the field contains cloud data from the machine the service is running on.'
  type: group
  fields:
    - name: instance.id
      level: extended
      type: keyword
      ignore_above: 1024
      description: Instance ID of the host machine.
      example: i-1234567890abcdef0
    - level: extended
      type: keyword
      ignore_above: 1024
      description: Availability zone in which this host is running.
      example: us-east-1c
    - name: account.id
      level: extended
      type: keyword
      ignore_above: 1024
      description: 'The cloud account or organization id used to identify different entities in a multi-tenant environment.

        Examples: AWS account id, Google Cloud ORG Id, or other unique identifier.'
      example: 666777888999
`

	fieldsTwoMissingNames = `- name: cloud
  title: Cloud
  group: 2
  description: Fields related to the cloud or infrastructure the events are coming from.
  footnote: 'Examples: If Metricbeat is running on an EC2 host and fetches data from its host, the cloud info contains the data about this machine. If Metricbeat runs on a remote machine outside the cloud and fetches data from a service running in the cloud, the field contains cloud data from the machine the service is running on.'
  type: group
  fields:
    - name: instance.id
      level: extended
      type: keyword
      ignore_above: 1024
      description: Instance ID of the host machine.
      example: i-1234567890abcdef0
    - level: extended
      type: keyword
      ignore_above: 1024
      description: Availability zone in which this host is running.
      example: us-east-1c
    - level: extended
      type: keyword
      ignore_above: 1024
      description: 'The cloud account or organization id used to identify different entities in a multi-tenant environment.

        Examples: AWS account id, Google Cloud ORG Id, or other unique identifier.'
      example: 666777888999
`
)

var indentTests = []struct {
	name string
	in   string
	want string
}{
	{
		name: "in_line_json_indent_array",
		in: `
    - e:
        s:
           [
             "key",
             "value"
           ]
`,
		want: `    - e:
        s: ["key", "value"]`,
	},
	{
		name: "in_line_json_indent_object",
		in: `
    - e:
        s:
           [
             {
               "key": "value"
             }
           ]
`,
		want: `    - e:
        s: [{"key": "value"}]`,
	},
}

// See https://github.com/goccy/go-yaml/issues/324
func TestFixIndent(t *testing.T) {
	for _, test := range indentTests {
		t.Run(test.name, func(t *testing.T) {
			file, err := parser.ParseBytes([]byte(test.in), parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse document: %v", err)
			}
			for _, doc := range file.Docs {
				ast.Walk(indentVisitor{}, doc)
			}
			got := file.String()

			if got != test.want {
				t.Errorf("unexpected result:\n--- got\n+++ want\n%s", cmp.Diff(got, test.want))
			}
		})
	}
}
