package agent

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func Assemble(task *DownloadTask) (err error) {
	log.Printf("starting assemble %s", task.Image)
	defer func() {
		if err != nil {
			fmt.Errorf("assemble error: %s", err.Error())
		} else {
			log.Printf("complete assemble %s", task.Image)
		}
	}()

	imageTarFile, err := os.Create(task.ImageTarPath)
	if err != nil {
		return
	}

	tw := tar.NewWriter(imageTarFile)
	defer tw.Close()

	repositoriesFile, err := os.Open(filepath.Join(task.TmpDir, "repositories"))
	if err != nil {
		return
	}

	info, err := repositoriesFile.Stat()
	if err != nil {
		return
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return
	}

	if err = tw.WriteHeader(header); err != nil {
		return
	}

	if _, err = io.Copy(tw, repositoriesFile); err != nil {
		return
	}

	for _, layerTarFilePath := range task.Torrents {
		layerTarFile, err := os.Open(layerTarFilePath)
		if err != nil {
			return fmt.Errorf("assemble error: %s", err.Error())
		}
		defer layerTarFile.Close()

		gr, err := gzip.NewReader(layerTarFile)
		if err != nil {
			return fmt.Errorf("assemble error: %s", err.Error())
		}
		defer gr.Close()

		tr := tar.NewReader(gr)

		for {
			header, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("assemble error: %s", err.Error())
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