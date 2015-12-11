package agent

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"os"
)

func Assemble(packagePaths []string, imageTarPath string) (err error) {
	imageTarFile, err := os.Create(imageTarPath)
	if err != nil {
		return
	}
	defer imageTarFile.Close()

	tw := tar.NewWriter(imageTarFile)
	defer tw.Close()

	for _, path := range packagePaths {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		gr, err := gzip.NewReader(file)
		if err != nil {
			return err
		}

		tr := tar.NewReader(gr)
		for {
			header, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			switch header.Typeflag {
			case tar.TypeDir:
				if err = tw.WriteHeader(header); err != nil {
					return err
				}
			case tar.TypeReg:
				if err = tw.WriteHeader(header); err != nil {
					return err
				}

				if _, err = io.Copy(tw, tr); err != nil {
					return err
				}
			default:
				return errors.New("unsupport tar header type")
			}
		}
	}

	return
}
