package appdist

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func writeFile(path string, data []byte, mode fs.FileMode, epoch time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return err
	}
	return os.Chtimes(path, epoch, epoch)
}

func copyFile(source, target string, mode fs.FileMode, epoch time.Time) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chtimes(target, epoch, epoch)
}

func archiveEntries(root string) ([]string, error) {
	var entries []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(entries)
	return entries, err
}

func writeTarGz(root, output, prefix string, epoch time.Time) error {
	entries, err := archiveEntries(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	gzipWriter, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		_ = file.Close()
		return err
	}
	gzipWriter.Header.ModTime = epoch
	gzipWriter.Header.OS = 3
	tarWriter := tar.NewWriter(gzipWriter)
	for _, rel := range entries {
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive entry %s is a symlink", rel)
		}
		name := rel
		if prefix != "" {
			name = strings.TrimSuffix(prefix, "/") + "/" + rel
		}
		header := &tar.Header{Name: name, Mode: int64(info.Mode().Perm()), ModTime: epoch, AccessTime: epoch, ChangeTime: epoch, Uid: 0, Gid: 0, Uname: "root", Gname: "root"}
		if info.IsDir() {
			header.Typeflag = tar.TypeDir
			header.Name += "/"
		} else {
			header.Typeflag = tar.TypeReg
			header.Size = info.Size()
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			input, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(tarWriter, input)
			_ = input.Close()
			if copyErr != nil {
				return copyErr
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Chtimes(output, epoch, epoch)
}

func writeZip(root, output, prefix string, epoch time.Time) error {
	entries, err := archiveEntries(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(file)
	zipTime := epoch
	if zipTime.Year() < 1980 {
		zipTime = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	for _, rel := range entries {
		path := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive entry %s is a symlink", rel)
		}
		name := rel
		if prefix != "" {
			name = strings.TrimSuffix(prefix, "/") + "/" + rel
		}
		if info.IsDir() {
			name += "/"
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = name
		header.Modified = zipTime
		header.Method = zip.Deflate
		header.SetMode(info.Mode())
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			input, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(entry, input)
			_ = input.Close()
			if copyErr != nil {
				return copyErr
			}
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Chtimes(output, epoch, epoch)
}
