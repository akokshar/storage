// +build linux

package main

import (
	"os"
	"syscall"
)

func getModDates(path string) (int64, int64, error) {
	fs, err := os.Stat(path)
	if err != nil {
		return -1, -1, nil
	}

	sysStat := fs.Sys().(*syscall.Stat_t)

	cDate := sysStat.Ctim.Sec
	mDate := sysStat.Mtim.Sec

	return cDate, mDate, nil
}
