package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var releaseMetadataFiles = []string{
	"README.md",
	"README.zh-CN.md",
	"CHANGELOG.md",
	"VERSION",
	"LICENSE",
}

func main() {
	library := flag.String("library", "", "path to the built shared library")
	archive := flag.String("archive", "", "path to the release archive to create")
	checksum := flag.String("checksum", "", "path to the checksum file to create")
	flag.Parse()

	if err := packageRelease(*library, *archive, *checksum); err != nil {
		fmt.Fprintf(os.Stderr, "package release: %v\n", err)
		os.Exit(1)
	}
}

func packageRelease(library, archive, checksum string) error {
	if library == "" {
		return errors.New("library is required")
	}
	if archive == "" {
		return errors.New("archive is required")
	}
	if _, err := os.Stat(library); err != nil {
		return fmt.Errorf("stat library: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		return fmt.Errorf("create archive directory: %w", err)
	}
	if err := writeArchive(library, archive); err != nil {
		return err
	}
	if checksum != "" {
		if err := writeChecksum(archive, checksum); err != nil {
			return err
		}
	}
	return nil
}

func writeArchive(library, archive string) error {
	out, err := os.Create(archive)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer func() {
		if errClose := out.Close(); errClose != nil {
			fmt.Fprintf(os.Stderr, "close archive file: %v\n", errClose)
		}
	}()

	zw := zip.NewWriter(out)
	if errAdd := addFileToZip(zw, library, filepath.Base(library)); errAdd != nil {
		return errAdd
	}
	for _, name := range releaseMetadataFiles {
		if _, errStat := os.Stat(name); errStat == nil {
			if errAdd := addFileToZip(zw, name, name); errAdd != nil {
				return errAdd
			}
		} else if !errors.Is(errStat, os.ErrNotExist) {
			return fmt.Errorf("stat metadata %s: %w", name, errStat)
		}
	}
	if errClose := zw.Close(); errClose != nil {
		return fmt.Errorf("close zip writer: %w", errClose)
	}
	return nil
}

func addFileToZip(zw *zip.Writer, source, name string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open %s: %w", source, err)
	}
	defer func() {
		if errClose := in.Close(); errClose != nil {
			fmt.Fprintf(os.Stderr, "close %s: %v\n", source, errClose)
		}
	}()
	info, err := in.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", source, err)
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("create zip header for %s: %w", source, err)
	}
	header.Name = filepath.ToSlash(name)
	header.Method = zip.Deflate
	writer, err := zw.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create zip entry for %s: %w", source, err)
	}
	if _, errCopy := io.Copy(writer, in); errCopy != nil {
		return fmt.Errorf("copy %s into zip: %w", source, errCopy)
	}
	return nil
}

func writeChecksum(archive, checksum string) error {
	in, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("open archive for checksum: %w", err)
	}
	defer func() {
		if errClose := in.Close(); errClose != nil {
			fmt.Fprintf(os.Stderr, "close archive for checksum: %v\n", errClose)
		}
	}()
	hash := sha256.New()
	if _, errCopy := io.Copy(hash, in); errCopy != nil {
		return fmt.Errorf("hash archive: %w", errCopy)
	}
	line := fmt.Sprintf("%s  %s\n", hex.EncodeToString(hash.Sum(nil)), filepath.Base(archive))
	if err := os.MkdirAll(filepath.Dir(checksum), 0o755); err != nil {
		return fmt.Errorf("create checksum directory: %w", err)
	}
	if err := os.WriteFile(checksum, []byte(line), 0o644); err != nil {
		return fmt.Errorf("write checksum: %w", err)
	}
	return nil
}
