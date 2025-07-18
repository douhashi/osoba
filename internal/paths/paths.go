package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// PathManager はosobaのファイルパスを管理するインターフェース
type PathManager interface {
	DataDir() string
	RunDir() string
	LogDir(repoIdentifier string) string
	PIDFile(repoIdentifier string) string
	EnsureDirectories() error
	AllPIDFiles() ([]string, error)
}

type pathManager struct {
	baseDir string
}

// NewPathManager は新しいPathManagerを作成します
func NewPathManager(baseDir string) PathManager {
	if baseDir == "" {
		baseDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "osoba")
	}
	return &pathManager{
		baseDir: baseDir,
	}
}

// DataDir はデータディレクトリのパスを返します
func (p *pathManager) DataDir() string {
	return p.baseDir
}

// RunDir はPIDファイルを格納するディレクトリのパスを返します
func (p *pathManager) RunDir() string {
	return filepath.Join(p.baseDir, "run")
}

// LogDir は指定されたリポジトリのログディレクトリのパスを返します
func (p *pathManager) LogDir(repoIdentifier string) string {
	sanitized := p.sanitizeIdentifier(repoIdentifier)
	return filepath.Join(p.baseDir, "logs", sanitized)
}

// PIDFile は指定されたリポジトリのPIDファイルのパスを返します
func (p *pathManager) PIDFile(repoIdentifier string) string {
	sanitized := p.sanitizeIdentifier(repoIdentifier)
	return filepath.Join(p.RunDir(), sanitized+".pid")
}

// EnsureDirectories は必要なディレクトリを作成します
func (p *pathManager) EnsureDirectories() error {
	dirs := []string{
		p.RunDir(),
		filepath.Join(p.baseDir, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// AllPIDFiles はすべてのPIDファイルのパスを返します
func (p *pathManager) AllPIDFiles() ([]string, error) {
	runDir := p.RunDir()
	entries, err := os.ReadDir(runDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var pidFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pid") {
			pidFiles = append(pidFiles, filepath.Join(runDir, entry.Name()))
		}
	}

	return pidFiles, nil
}

// sanitizeIdentifier はファイルシステムで安全な識別子に変換します
func (p *pathManager) sanitizeIdentifier(identifier string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		".", "_",
		" ", "_",
	)
	return replacer.Replace(identifier)
}
