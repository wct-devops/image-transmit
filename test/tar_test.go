package test

import (
	"archive/tar"
	"io"
	"testing"
	"os"
)

func TestTar(t *testing.T) {

	out, _ := os.Create("test.tar")

	var tw *tar.Writer
	tw = tar.NewWriter(out)

	/*
	folder := &tar.Header{
		Name: "folder",
		Size: 0,
		Mode: tar.TypeDir,
	}
	tw.WriteHeader(folder)
	*/

	file := &tar.Header{
		Name: "folder/squashfs.console.log",
		Size: 282,
		Mode: tar.TypeReg,
	}
	tw.WriteHeader(file)

	logfile, _ := os.Open("squashfs.console.log")
	io.Copy(tw, logfile)

	tw.Close()
	out.Close()

}
