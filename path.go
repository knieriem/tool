package tool

import (
	"os"
	"path/filepath"
)

func FilepathRelIfShorter(fileName string) string {
	p, err := filepath.Abs(fileName)
	if err != nil {
		p = fileName
	}

	wd, err := os.Getwd()
	if err == nil {
		rel, err := filepath.Rel(wd, p)
		if err == nil && len(rel) < len(fileName) {
			fileName = rel
		}
	}
	return fileName
}
