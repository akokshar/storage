// +build linux

package meta

import (
	"os"
	"syscall"
)

func getFileTimeStamps(fi os.FileInfo) (int64, int64) {
	sysStat := fi.Sys().(*syscall.Stat_t)

	cDate := sysStat.Ctim.Sec
	mDate := sysStat.Mtim.Sec

	return cDate, mDate
}
