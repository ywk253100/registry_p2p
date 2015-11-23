package utils

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func FileExist(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func Gzip(src, dst string) (err error) {
	s, err := os.Open(src)
	if err != nil {
		return
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return
	}
	defer d.Close()

	w := gzip.NewWriter(d)
	defer w.Close()

	if _, err = io.Copy(w, s); err != nil {
		return
	}
	return
}

func Copy(src, dst string) (err error) {
	info, err := os.Stat(src)
	if err != nil {
		return
	}

	if info.IsDir() {
		return copyDir(src, dst)
	}

	return copyFile(src, dst)
}

func copyFile(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	name := info.Name()

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(filepath.Join(dst, name))
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	err = dstFile.Chmod(info.Mode())
	if err != nil {
		return err
	}

	return nil
}

func copyDir(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	name := info.Name()

	err = os.MkdirAll(filepath.Join(dst, name), info.Mode())
	if err != nil {
		return err
	}

	infos, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, info := range infos {
		s := filepath.Join(src, info.Name())
		d := filepath.Join(filepath.Join(dst, name))

		if info.IsDir() {
			err = copyDir(s, d)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(s, d)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
