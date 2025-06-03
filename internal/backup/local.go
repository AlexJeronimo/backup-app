package backup

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func CopyFiles(sourcePath, destPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Warning: Source file does not finded: %s", sourcePath)
			return fmt.Errorf("source file does not exist: %w", err)
		} else {
			return fmt.Errorf("source file open error '%s': %w", sourcePath, err)
		}

	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("receive source file information error '%s': %w", sourcePath, err)
	}

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, sourceInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("create destination directory error '%s': %w", destDir, err)
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("destination file create error '%s': %w", destPath, err)
	}
	defer destFile.Close()

	bytesCopied, err := io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("data copy error from '%s' to '%s': %w", sourcePath, destPath, err)
	}

	if err := os.Chtimes(destPath, sourceInfo.ModTime(), sourceInfo.ModTime()); err != nil {
		log.Printf("Warning: Can't copy modification time for '%s': %v", destPath, err)
	}

	log.Printf("File successfully copy: '%s' -> '%s' (%d bytes)\n", sourcePath, destPath, bytesCopied)
	return nil
}

// TODO: Feature realization for VSS copy
func CopyFilesWithVSS(sourcePath, destPath string) error {
	log.Printf("Warning: Function CopyFilesWithVSS does not realized yet. Using simple copy without VSS for '%s'.", sourcePath)
	return CopyFiles(sourcePath, destPath)
}

// TODO: Feature realization for ACL copy
func CopyAttributesAndACL(sourcePath, destPath string) error {
	log.Printf("Warning: Function CopyAttributesAndACL does not realized yet for '%s'.", sourcePath)
	return nil
}
