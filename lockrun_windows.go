package lockrun

import (
	"golang.org/x/sys/windows"
	"io/ioutil"
	"os"
	"path/filepath"
	"fmt"
	"unsafe"
)

func CheckCanLock(fname string) error {

	stat, err := os.Stat(fname)
	if err == nil && stat.IsDir() {
		return fmt.Errorf("It is directory path.")
	}

	dir := filepath.Dir(fname)
	baseFName := filepath.Base(fname)
	tmpFile, err := ioutil.TempFile(dir, baseFName)
	if err != nil {
		return err
	}

	tmpFile.Close()
	os.Remove(tmpFile.Name())
	return nil
}

var handleLockFile windows.Handle = windows.InvalidHandle

func WaitAndLock(fName string) bool {
	fnameUTF16, err := windows.UTF16FromString(fName)
	if err != nil {
		panic(err)
	}
	handleLockFile, err = windows.CreateFile((*uint16)(unsafe.Pointer(&fnameUTF16[0])), windows.GENERIC_WRITE, 0, nil, windows.CREATE_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	return err == nil
}

func Unlock(){
	if handleLockFile != windows.InvalidHandle {
		windows.Close(handleLockFile)
	}
	handleLockFile = windows.InvalidHandle
}