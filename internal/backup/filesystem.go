package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func CopyFile(src, dst string) (written int64, err error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, fmt.Errorf("error getting source file information '%s': %w", src, err)
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("source file '%s' is not a regular file", src)
	}

	destFileStat, err := os.Stat(dst)
	if err == nil {
		if sourceFileStat.ModTime().Equal(destFileStat.ModTime()) && sourceFileStat.Size() == destFileStat.Size() {
			return 0, fmt.Errorf("file '%s' not changed and already exists in '%s', copying scipped", src, dst)
		}
	} else if !os.IsNotExist(err) {
		return 0, fmt.Errorf("error getting information about destination file '%s': %w", dst, err)
	}
	source, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("error opening source file '%s': %w", src, err)
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, fmt.Errorf("error creating destination file '%s': %w", dst, err)
	}
	defer destination.Close()

	written, err = io.Copy(destination, source)
	if err != nil {
		return written, fmt.Errorf("error copy file content from '%s' to '%s': %w", src, dst, err)
	}

	if err := os.Chmod(dst, sourceFileStat.Mode()); err != nil {
		return written, fmt.Errorf("error getting permissions for '%s': %w", dst, err)
	}
	if err := os.Chtimes(dst, time.Now(), sourceFileStat.ModTime()); err != nil {
		return written, fmt.Errorf("error setting modification time for '%s': %w", dst, err)
	}

	return written, nil
}

func CopyDir(src, dst string, totalCopiedFiles *int, totalCopiedBytes *int64) error {
	src = filepath.Clean(src)

	sourceInfo, err := os.Stat(src)
	if err != nil {
		fmt.Errorf("error getting source directory information '%s': %w", src, err)
	}

	if !sourceInfo.IsDir() {
		return fmt.Errorf("error source '%s' is not directory", src)
	}

	if err := os.MkdirAll(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("erro creating destination directory '%s': %w", dst, err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error reading directory '%s': %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath, totalCopiedFiles, totalCopiedBytes); err != nil {
				return fmt.Errorf("error recursive copying directory '%s': %w", srcPath, err)
			}
		} else {
			written, err := CopyFile(srcPath, dstPath)
			if err != nil {
				if err.Error() == fmt.Sprintf("file '%s' not changed and already exists in '%s', copying scipped", srcPath, dstPath) {
					// fmt.Printf("DEBUG: %s\n", err.Error()) //For debug purposes
				} else {
					return fmt.Errorf("error copy file '%s' in '%s': %w", srcPath, dstPath, err)
				}
			} else {
				*totalCopiedFiles++
				*totalCopiedBytes += written
			}
		}
	}

	return nil
}

func GetDirSIze(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("error during counting directory size '%s': %w", path, err)
	}

	return size, nil
}
