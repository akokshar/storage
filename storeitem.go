package main

import (
	"path"
	"strings"
	"errors"
	"os"
	"io/ioutil"
)

type StoreItem interface {
	getPath() string

	GetJson(opts map[string]string) (string, error)
	CreateFile(content []byte) error
	CreateDir() error
	Delete() error
}

type storeItem struct {
	filepath string
}

type storeItemInfo struct {
	Name		string		`json:"name"`
	IsDirectory	bool		`json:"directory"`
	ModDate		int64		`json:"create_date"`
	Size		int64		`json:"size"`
}

type storeDir struct {
	storeItemInfo
	content struct {
		Offset	int				`json:"offset"`
		Count	int				`json:"count"`
		files	[]storeItemInfo	`json:"files"`
	}							`json:"content"`
}

type storeFile struct {
	storeItemInfo
	content struct {
		b64str string			`json:"b_64_str"`
	} 							`json:"content"`
}

func NewStoreItem(basepath string, itempath string) (StoreItem, error) {
	filepath := path.Join(basedir, itempath)
	if strings.HasPrefix(filepath, basedir) {
		return &storeItem{filepath}, nil
	}
	return nil, errors.New("Wrong path")
}

func (i *storeItem) getPath() string {
	return i.filepath
}

func (i *storeItem) GetJson(opts map[string]string) (string, error) {
	fileInfo, err := os.Stat(i.getPath())
	if err != nil {
		return "", err
	}

	if fileInfo.IsDir() {
		files, err := ioutil.ReadDir(i.getPath())
		if err != nil {
			return "", err
		}
		size := int64(len(files))

		offset := opts["offset"]
		count := opts["count"]

		fInfo := &storeDir{
			storeItemInfo{
				Name: fileInfo.Name(),
				IsDirectory: fileInfo.IsDir(),
				ModDate: fileInfo.ModTime().Unix(),
				Size: size,
			},
			{
				Offset: 0,
				Count: 0,
				files: nil,
			},
		}
	} else {
		return "", nil
	}
}

func (i *storeItem) CreateFile(content []byte) error {
	file, err := os.Create(i.getPath());
	if err != nil {
		return err
	}
	defer file.Close()

	if content != nil {
		file.Write(content)
	}
	return nil
}

func (i *storeItem) CreateDir() error {
	if err := os.Mkdir(i.getPath(), os.ModeDir); err != nil {
		return err
	}
	return nil
}

func (i *storeItem) Delete() error {
	if err := os.Remove(i.getPath()); err != nil {
		return err
	}
	return nil
}