package webui

import (
	"archive/zip"
	"bytes"
	"io"
	"io/fs"
	"path"
	"time"
)

type unzippedFS struct {
	files map[string]*unzippedFile
}

type unzippedFile struct {
	data    []byte
	modTime time.Time
}

func (f *unzippedFS) Open(name string) (fs.File, error) {
	file, ok := f.files[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &unzippedFileHandle{
		unzippedFile: file,
		name:         name,
		reader:       bytes.NewReader(file.data),
	}, nil
}

type unzippedFileHandle struct {
	*unzippedFile
	name   string
	reader *bytes.Reader
}

func (f *unzippedFileHandle) Stat() (fs.FileInfo, error) {
	return &unzippedFileInfo{
		name:    path.Base(f.name),
		size:    int64(len(f.data)),
		modTime: f.modTime,
	}, nil
}

func (f *unzippedFileHandle) Read(b []byte) (int, error) {
	return f.reader.Read(b)
}

func (f *unzippedFileHandle) Close() error {
	return nil
}

type unzippedFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (i *unzippedFileInfo) Name() string       { return i.name }
func (i *unzippedFileInfo) Size() int64        { return i.size }
func (i *unzippedFileInfo) Mode() fs.FileMode  { return 0444 }
func (i *unzippedFileInfo) ModTime() time.Time { return i.modTime }
func (i *unzippedFileInfo) IsDir() bool        { return false }
func (i *unzippedFileInfo) Sys() any           { return nil }

func UnzipToFS(zipData []byte) fs.FS {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		panic(err)
	}

	files := make(map[string]*unzippedFile)
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			panic(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			panic(err)
		}
		files[file.Name] = &unzippedFile{
			data:    data,
			modTime: file.Modified,
		}
	}

	return &unzippedFS{files: files}
}
