package tmux

import (
	"github.com/douhashi/osoba/internal/logger"
)

// packageState はパッケージレベルの状態を保持
type packageState struct {
	logger logger.Logger
}

// pkg はパッケージレベルの状態インスタンス
var pkg = &packageState{}

// SetLogger はパッケージ全体で使用するロガーを設定
func SetLogger(l logger.Logger) {
	pkg.logger = l
}

// GetLogger は設定されているロガーを取得
func GetLogger() logger.Logger {
	return pkg.logger
}
