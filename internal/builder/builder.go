package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Language represents a supported programming language
type Language struct {
	Name           string
	Extensions     []string
	BuildCommand   []string
	TestCommand    []string
	RunCommand     []string
	Compiler       string
	PackageManager string
	LSPServer      string
}

// SupportedLanguages defines all languages from Phase 1 specification
var SupportedLanguages = map[string]Language{
	"go": {
		Name:           "Go",
		Extensions:     []string{".go"},
		BuildCommand:   []string{"go", "build", "./..."},
		TestCommand:    []string{"go", "test", "./..."},
		RunCommand:     []string{"go", "run"},
		Compiler:       "go",
		PackageManager: "go mod",
		LSPServer:      "gopls",
	},
	"rust": {
		Name:           "Rust",
		Extensions:     []string{".rs"},
		BuildCommand:   []string{"cargo", "build"},
		TestCommand:    []string{"cargo", "test"},
		RunCommand:     []string{"cargo", "run"},
		Compiler:       "rustc",
		PackageManager: "cargo",
		LSPServer:      "rust-analyzer",
	},
	"python": {
		Name:           "Python",
		Extensions:     []string{".py"},
		BuildCommand:   []string{"python", "-m", "build"},
		TestCommand:    []string{"pytest"},
		RunCommand:     []string{"python"},
		Compiler:       "python3",
		PackageManager: "pip",
		LSPServer:      "pylsp",
	},
	"javascript": {
		Name:           "JavaScript",
		Extensions:     []string{".js", ".mjs"},
		BuildCommand:   []string{"npm", "run", "build"},
		TestCommand:    []string{"npm", "test"},
		RunCommand:     []string{"node"},
		Compiler:       "node",
		PackageManager: "npm",
		LSPServer:      "typescript-language-server",
	},
	"typescript": {
		Name:           "TypeScript",
		Extensions:     []string{".ts", ".tsx"},
		BuildCommand:   []string{"tsc"},
		TestCommand:    []string{"npm", "test"},
		RunCommand:     []string{"ts-node"},
		Compiler:       "tsc",
		PackageManager: "npm",
		LSPServer:      "typescript-language-server",
	},
	"java": {
		Name:           "Java",
		Extensions:     []string{".java"},
		BuildCommand:   []string{"mvn", "compile"},
		TestCommand:    []string{"mvn", "test"},
		RunCommand:     []string{"java"},
		Compiler:       "javac",
		PackageManager: "maven",
		LSPServer:      "jdtls",
	},
	"cpp": {
		Name:           "C++",
		Extensions:     []string{".cpp", ".cc", ".cxx", ".hpp", ".h"},
		BuildCommand:   []string{"cmake", "--build", "build"},
		TestCommand:    []string{"ctest"},
		RunCommand:     []string{"./main"},
		Compiler:       "g++",
		PackageManager: "vcpkg",
		LSPServer:      "clangd",
	},
	"c": {
		Name:           "C",
		Extensions:     []string{".c", ".h"},
		BuildCommand:   []string{"make"},
		TestCommand:    []string{"make", "test"},
		RunCommand:     []string{"./main"},
		Compiler:       "gcc",
		PackageManager: "make",
		LSPServer:      "clangd",
	},
	"php": {
		Name:           "PHP",
		Extensions:     []string{".php", ".phtml", ".php3", ".php4", ".php5", ".phps"},
		BuildCommand:   []string{"composer", "install"},
		TestCommand:    []string{"./vendor/bin/phpunit"},
		RunCommand:     []string{"php"},
		Compiler:       "php",
		PackageManager: "composer",
		LSPServer:      "phpactor",
	},
}

// DetectLanguage determines the language based on file extension
func DetectLanguage(filePath string) (Language, error) {
	ext := filepath.Ext(filePath)

	for _, lang := range SupportedLanguages {
		for _, langExt := range lang.Extensions {
			if ext == langExt {
				return lang, nil
			}
		}
	}

	return Language{}, fmt.Errorf("unsupported file extension: %s", ext)
}

// detectProjectLanguage detects the primary language of a project
func detectProjectLanguage(projectPath string) (Language, error) {
	// Check for language-specific files in order of priority
	languageFiles := map[string]string{
		"go.mod":           "go",
		"Cargo.toml":       "rust",
		"package.json":     "javascript",
		"tsconfig.json":    "typescript",
		"pom.xml":          "java",
		"CMakeLists.txt":   "cpp",
		"Makefile":         "c",
		"makefile":         "c",
		"composer.json":    "php",
		"requirements.txt": "python",
		"setup.py":         "python",
		"pyproject.toml":   "python",
	}

	for file, langKey := range languageFiles {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			if lang, exists := SupportedLanguages[langKey]; exists {
				return lang, nil
			}
		}
	}

	return Language{}, fmt.Errorf("could not detect project language in %s", projectPath)
}

// Build executes the build command for the detected language
func Build(projectPath string) ([]byte, error) {
	// Try to detect language from project structure
	lang, err := detectProjectLanguage(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project language: %w", err)
	}

	return BuildWithLanguage(projectPath, lang)
}

// BuildWithLanguage builds a project with a specific language
func BuildWithLanguage(projectPath string, lang Language) ([]byte, error) {
	cmd := exec.Command(lang.BuildCommand[0], lang.BuildCommand[1:]...)
	cmd.Dir = projectPath
	return cmd.CombinedOutput()
}

// BuildGo maintains backward compatibility
func BuildGo() ([]byte, error) {
	return Build(".")
}

// BuildPHP builds a PHP project
func BuildPHP(projectPath string) ([]byte, error) {
	lang := SupportedLanguages["php"]
	return BuildWithLanguage(projectPath, lang)
}

// BuildC builds a C project
func BuildC(projectPath string) ([]byte, error) {
	lang := SupportedLanguages["c"]
	return BuildWithLanguage(projectPath, lang)
}

// TestPHP runs PHP tests
func TestPHP(projectPath string) ([]byte, error) {
	lang := SupportedLanguages["php"]
	cmd := exec.Command(lang.TestCommand[0], lang.TestCommand[1:]...)
	cmd.Dir = projectPath
	return cmd.CombinedOutput()
}

// TestC runs C tests
func TestC(projectPath string) ([]byte, error) {
	lang := SupportedLanguages["c"]
	cmd := exec.Command(lang.TestCommand[0], lang.TestCommand[1:]...)
	cmd.Dir = projectPath
	return cmd.CombinedOutput()
}

// RunPHP executes a PHP file
func RunPHP(projectPath string, fileName string) ([]byte, error) {
	lang := SupportedLanguages["php"]
	cmd := exec.Command(lang.RunCommand[0], fileName)
	cmd.Dir = projectPath
	return cmd.CombinedOutput()
}

// RunC executes a compiled C program
func RunC(projectPath string, executableName string) ([]byte, error) {
	cmd := exec.Command("./" + executableName)
	cmd.Dir = projectPath
	return cmd.CombinedOutput()
}

func ApplyFix(filePath string, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

func ParseError(output string) (string, string) {
	// Define error patterns for different languages
	errorPatterns := []struct {
		pattern string
		fileIdx int
		lineIdx int
	}{
		// Go errors: ./main.go:10:5: error message
		{`(?m)(.*?):(\d+):(\d+): (.*)`, 1, 2},

		// C/C++ GCC errors: main.c:10:5: error: message
		{`(?m)(.*?):(\d+):(\d+): error: (.*)`, 1, 2},

		// C/C++ Clang errors: main.c:10:5: error: message
		{`(?m)(.*?):(\d+):(\d+): error: (.*)`, 1, 2},

		// PHP errors: PHP Parse error: syntax error in /path/file.php on line 10
		{`(?m)PHP .*?error.*? in (.*?) on line (\d+)`, 1, 2},

		// PHP Fatal errors: Fatal error: message in /path/file.php on line 10
		{`(?m)Fatal error:.*? in (.*?) on line (\d+)`, 1, 2},

		// Make errors: make: *** [target] Error in file.c:10
		{`(?m).*?\*\*\* \[.*?\] Error.*? in (.*?):(\d+)`, 1, 2},

		// Generic file:line pattern
		{`(?m)(.*?):(\d+): (.*)`, 1, 2},
	}

	for _, pattern := range errorPatterns {
		re := regexp.MustCompile(pattern.pattern)
		matches := re.FindStringSubmatch(output)

		if len(matches) > pattern.lineIdx {
			filePath := strings.TrimSpace(matches[pattern.fileIdx])
			lineNumber := matches[pattern.lineIdx]
			return filePath, lineNumber
		}
	}

	return "", ""
}

func ExtractCode(response string) string {
	// Define code block patterns for different languages
	codePatterns := []string{
		"(?s)```go\n(.*?)```",
		"(?s)```c\n(.*?)```",
		"(?s)```cpp\n(.*?)```",
		"(?s)```c\\+\\+\n(.*?)```",
		"(?s)```php\n(.*?)```",
		"(?s)```python\n(.*?)```",
		"(?s)```javascript\n(.*?)```",
		"(?s)```typescript\n(.*?)```",
		"(?s)```java\n(.*?)```",
		"(?s)```rust\n(.*?)```",
		"(?s)```\n(.*?)```", // Generic code block
	}

	for _, pattern := range codePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(response)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	return response
}

func GenerateDiff(filePath string, newContent string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(content), newContent, false)

	return dmp.DiffPrettyText(diffs), nil
}
