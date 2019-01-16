// +build darwin

package meta

import (
	"os"
	"syscall"
)

func getFileTimeStamps(fi os.FileInfo) (int64, int64) {
	sysStat := fi.Sys().(*syscall.Stat_t)

	cDate := sysStat.Ctimespec.Sec
	mDate := sysStat.Mtimespec.Sec

	return cDate, mDate
}
