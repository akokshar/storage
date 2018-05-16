package main

import (
	"path"
	"strings"
	"errors"
	"os"
	"io/ioutil"
	"encoding/json"
	"strconv"
	"encoding/base64"
)

type FileStoreItem interface {
	getPath() string
	createFileStoreItemInfo() (*fileStoreItemInfo, error)

	GetJson(opts map[string]string) ([]byte, error)
	CreateFile(content []byte) error
	CreateDir() error
	Delete() error
}

type fileStoreItem struct {
	filepath string
}

type fileStoreItemInfo struct {
	Name		string	`json:"name"`
	IsDirectory	bool	`json:"directory"`
	ModDate		int64	`json:"create_date"`
	Size		int64	`json:"size"`

	files		[]os.FileInfo
}

const DirListOffsetOptName = "offset"
const defaultDirListOffset = 0
const DirListCountOptName = "count"
const defaultDirListCount = 10

type fileStoreDirContent struct {
	Offset	int					`json:"offset"`
	Count	int					`json:"count"`
	Files	[]*fileStoreItemInfo	`json:"files"`
}

type storeDir struct {
	fileStoreItemInfo
	Content fileStoreDirContent	`json:"content"`
}

type fileStoreFileContent struct {
	B64str string	`json:"b_64_str"`
}

type storeFile struct {
	fileStoreItemInfo
	Content fileStoreFileContent	`json:"content"`
}

func InitFileStoreItem(basepath string, itempath string) (FileStoreItem, error) {
	filepath := path.Join(basepath, itempath)
	if strings.HasPrefix(filepath, basepath) {
		return &fileStoreItem{filepath}, nil
	}
	return nil, errors.New("Wrong path")
}

func (storeItem *fileStoreItem) getPath() string {
	return storeItem.filepath
}

func (storeItem *fileStoreItem) createFileStoreItemInfo() (*fileStoreItemInfo, error) {
	var size int64
	var files []os.FileInfo
	fileInfo, err := os.Stat(storeItem.getPath())

	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		files, _ = ioutil.ReadDir(storeItem.getPath())
		size = int64(len(files))
	} else {
		size = fileInfo.Size()
	}

	itemInfo := &fileStoreItemInfo{
		Name:fileInfo.Name(),
		IsDirectory:fileInfo.IsDir(),
		ModDate:fileInfo.ModTime().Unix(),
		Size:size,
		files:files,
	}

	return itemInfo, nil
}

func (storeItem *fileStoreItem) GetJson(opts map[string]string) ([]byte, error) {
	itemInfo, err := storeItem.createFileStoreItemInfo()
	if err != nil {
		return nil, err
	}

	var result interface{}

	if itemInfo.IsDirectory {
		var offset int
		var count int

		if offset, err = strconv.Atoi(opts[DirListOffsetOptName]); err != nil {
			offset = defaultDirListOffset
		}

		if count, err = strconv.Atoi(opts[DirListCountOptName]); err != nil {
			count = defaultDirListCount
		}

		if tailLen := len(itemInfo.files) - offset; tailLen < count {
			count = tailLen
		}

		dir := &storeDir{
			*itemInfo,
			fileStoreDirContent{
				Offset: offset,
				Count: count,
				Files: make([]*fileStoreItemInfo, count),
			},
		}

		for i := 0; i < count; i++ {
			childItem, _ := InitFileStoreItem(storeItem.getPath(), itemInfo.files[i + offset].Name())
			childItemInfo, _ := childItem.createFileStoreItemInfo()
			dir.Content.Files[i] =  childItemInfo
		}

		result = dir
	} else {
		if file, err := os.Open(storeItem.filepath); err == nil {
			buf := make([]byte, itemInfo.Size)
			if _, err := file.Read(buf); err == nil {
				result = &storeFile{
					*itemInfo,
					fileStoreFileContent{
						B64str:base64.StdEncoding.EncodeToString(buf),
					},
				}
			}
		}
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return jsonResult, nil
}

func (storeItem *fileStoreItem) CreateFile(content []byte) error {
	file, err := os.Create(storeItem.getPath());
	if err != nil {
		return err
	}
	defer file.Close()

	if content != nil {
		file.Write(content)
	}
	return nil
}

func (storeItem *fileStoreItem) CreateDir() error {
	if err := os.Mkdir(storeItem.getPath(), os.ModeDir); err != nil {
		return err
	}
	return nil
}

func (storeItem *fileStoreItem) Delete() error {
	if err := os.Remove(storeItem.getPath()); err != nil {
		return err
	}
	return nil
}