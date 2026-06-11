package s3

import (
	"path/filepath"
	"strings"

	"aws-terminal/internal/config"
	domains3 "aws-terminal/internal/domain/s3"
)

func (p *S3Page) savePreferences() {
	if p.preferenceStore == nil {
		return
	}
	_ = p.preferenceStore.Save(p.preferences)
}

func (p *S3Page) rememberSource(source domains3.SourceSelection) {
	dir := strings.TrimSpace(source.Path)
	if source.Kind == domains3.SourceKindFile {
		dir = filepath.Dir(dir)
	}
	if dir == "." || dir == "" {
		return
	}
	p.preferences.S3SourceDirectory = dir
	p.savePreferences()
}

func (p *S3Page) rememberPrefix(prefix string) {
	p.preferences = config.RememberRecentPrefix(p.preferences, prefix)
	p.savePreferences()
}
