package updater

import (
	"os"
	"path/filepath"
	"testing"
)

// testWorkflowCase is a reusable test structure for workflow parsing tests
type testWorkflowCase struct {
	name     string
	content  string
	wantRefs int // number of action references expected
	wantErr  bool
}

// testWorkflowParsing is a helper function to test parsing workflow content
func testWorkflowParsing(t *testing.T, tc testWorkflowCase) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp dir: %v", err)
		}
	}(tempDir)

	// Set secure permissions on temp directory
	if err := os.Chmod(tempDir, 0750); err != nil {
		t.Fatalf("Failed to set temp dir permissions: %v", err)
	}

	// Create scanner with temp directory as base
	scanner := NewScanner(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "workflow.yml")
	err = os.WriteFile(testFile, []byte(tc.content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse action references
	refs, err := scanner.ParseActionReferences(testFile)
	if tc.wantErr {
		if err == nil {
			t.Error("Expected error, got nil")
		}
		return
	}
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(refs) != tc.wantRefs {
		t.Errorf("Expected %d references, got %d", tc.wantRefs, len(refs))
	}
}

func TestParseActionReferencesEdgeCases(t *testing.T) {
	// Define test cases for action references edge cases
	tests := []testWorkflowCase{
		{
			name: "nested steps with multiple uses",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3
      - name: Nested steps
        uses: actions/setup-node@2028fbc5c25fe9cf00d9f06a71cc4710d4507903
        steps:
          - actions/setup-python@83679a892e2d95755f2dac6acb0bfd1e9ac5d548`,
			wantRefs: 3,
			wantErr:  false,
		},
		{
			name: "valid yaml with no actions",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Hello"`,
			wantRefs: 0,
			wantErr:  false,
		},
		{
			name:     "empty file with valid yaml",
			content:  `{}`,
			wantRefs: 0,
			wantErr:  false,
		},
		{
			name:     "mixed line endings",
			content:  "name: Test Workflow\r\non: [push]\njobs:\r\n  test:\n    runs-on: ubuntu-latest\r\n    steps:\r\n      - uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3\n",
			wantRefs: 1,
			wantErr:  false,
		},
		{
			name: "comments in various positions",
			content: `# Header comment
name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      # Before action
      - uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3  # Inline comment
      # Between actions
      - uses: actions/setup-node@2028fbc5c25fe9cf00d9f06a71cc4710d4507903
      # After actions`,
			wantRefs: 2,
			wantErr:  false,
		},
		{
			name: "action reference with special characters in version",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3.3.4-beta.1+meta`,
			wantRefs: 1,
			wantErr:  false,
		},
		{
			name: "multiple jobs with same action",
			content: `name: Test Workflow
on: [push]
jobs:
  test1:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3
  test2:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3`,
			wantRefs: 2,
			wantErr:  false,
		},
		{
			name: "complex yaml with anchors and aliases",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps: &steps
      - uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3
  deploy:
    runs-on: ubuntu-latest
    steps: *steps`,
			wantRefs: 2,
			wantErr:  false,
		},
		{
			name: "unicode characters in workflow",
			content: `name: 测试工作流
on: [push]
jobs:
  テスト:
    runs-on: ubuntu-latest
    steps:
      - name: 检查代码
        uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3`,
			wantRefs: 1,
			wantErr:  false,
		},
		{
			name: "minimal valid workflow",
			content: `on: [push]
jobs:
  a:
    runs-on: a
    steps:
      - uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3`,
			wantRefs: 1,
			wantErr:  false,
		},
	}

	// Run tests using our helper function
	for _, tc := range tests {
		tc := tc // Create a local copy to avoid issues with closures
		t.Run(tc.name, func(t *testing.T) {
			testWorkflowParsing(t, tc)
		})
	}
}

func TestParseNodeEdgeCases(t *testing.T) {
	// Define test cases for node parsing edge cases
	tests := []testWorkflowCase{
		{
			name: "deeply nested uses",
			content: `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: outer
        uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3
        with:
          nested:
            steps:
              - uses: actions/setup-node@2028fbc5c25fe9cf00d9f06a71cc4710d4507903`,
			wantRefs: 2,
			wantErr:  false,
		},
		{
			name: "uses in matrix",
			content: `name: Test
on: [push]
jobs:
  test:
    strategy:
      matrix:
        action: ['actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3', 'actions/setup-node@2028fbc5c25fe9cf00d9f06a71cc4710d4507903']
    runs-on: ubuntu-latest
    steps:
      - uses: ${{ matrix.action }}`,
			wantRefs: 1, // Should only count direct uses
			wantErr:  false,
		},
		{
			name: "uses as plain string",
			content: `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Echo action
        run: |
          echo "uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3"`,
			wantRefs: 0,
			wantErr:  false,
		},
		{
			name: "uses in conditional",
			content: `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - if: ${{ github.event_name == 'push' }}
        uses: actions/checkout@1af3b93b6815bc44a9784bd300feb67ff0d1eeb3`,
			wantRefs: 1,
			wantErr:  false,
		},
	}

	// Run tests using our helper function
	for _, tc := range tests {
		tc := tc // Create a local copy to avoid issues with closures
		t.Run(tc.name, func(t *testing.T) {
			testWorkflowParsing(t, tc)
		})
	}
}
