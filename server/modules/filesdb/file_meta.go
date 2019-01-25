package filesdb

import (
	"errors"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	contentTypeDirectory = "folder"
)

type fileItem struct {
	path string
	fi   os.FileInfo
}

type fileMeta struct {
	ID    int64  `json:"id"`
	Size  int64  `json:"size"`
	MDate int64  `json:"mdate"`
	CDate int64  `json:"cdate"`
	Name  string `json:"name"`
	CType string `json:"ctype"`
}

type dirMeta struct {
	Offset int         `json:"offset"`
	Files  []*fileMeta `json:"files"`
}

// CreateFileItem initialize new fileItem
func createFileItem(path string) (*fileItem, error) {
	fi, err := os.Stat(path)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	if (!fi.Mode().IsDir() && !fi.Mode().IsRegular()) || strings.HasPrefix(".", fi.Name()) {
		return nil, errors.New("Not a directory, regular file or is hidden")
	}

	f := &fileItem{
		path: path,
		fi:   fi,
	}
	return f, nil
}

func (f *fileItem) GetItemPath() string {
	return f.path
}

func (f *fileItem) GetItemFileInfo() os.FileInfo {
	return f.fi
}

func (f *fileItem) fileMeta() *fileMeta {
	fm := new(fileMeta)
	fm.Name = f.fi.Name()
	if f.fi.IsDir() {
		fm.Size = 0
		fm.CType = contentTypeDirectory
	} else {
		fm.Size = f.fi.Size()

		f, err := os.Open(f.path)
		if err == nil {
			defer f.Close()
			buffer := make([]byte, 512)
			if count, _ := f.Read(buffer); count < 512 {
				fm.CType = mime.TypeByExtension(filepath.Ext(fm.Name))
			} else {
				fm.CType = http.DetectContentType(buffer)
			}
		}
		if fm.CType == "" {
			fm.CType = "application/octet-stream"
		}
	}
	fm.CDate, fm.MDate = getFileTimeStamps(f.fi)

	return fm
}
