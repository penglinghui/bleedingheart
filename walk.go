package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const BlockSize = 1024 * 1024

func (f *BhFile) Dump() {
	fmt.Printf("%s\n", BytesToString(f.Name))
	for _, b := range f.Blocks {
		fmt.Printf("  %dB @ %d: %x\n", *b.Length, *b.Offset, b.Hash)
	}
	fmt.Println()
}

func isTempName(name string) bool {
	return strings.HasPrefix(path.Base(name), ".bh.")
}

func tempName(name string, modified int64) string {
	tdir := path.Dir(name)
	tname := fmt.Sprintf(".bh.%s.%d", path.Base(name), modified)
	return path.Join(tdir, tname)
}

func genWalker(res *[]*BhFile) filepath.WalkFunc {
	return func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if isTempName(p) {
			return nil
		}

		if info.Mode()&os.ModeType == 0 {
			rn, err := filepath.Rel(g_Model.dir, p)
			if err != nil {
				return err
			}

			fi, err := os.Stat(p)
			if err != nil {
				return err
			}
			modified := fi.ModTime().Unix()

			hf, ok := g_Model.LocalFile(rn)
			if ok && *hf.Modified == modified {
				// No change
				*res = append(*res, hf)
			} else {
				fd, err := os.Open(p)
				if err != nil {
					return err
				}
				defer fd.Close()

				blocks, err := Blocks(fd, BlockSize)
				if err != nil {
					return err
				}
				flags := uint32(info.Mode())
				f := BhFile{
					Name:     StringToBytes(rn),
					Flags:    &flags,
					Modified: &modified,
					Blocks:   blocks,
				}
				*res = append(*res, &f)
			}
		}

		return nil
	}
}

func Walk() []*BhFile {
	var files []*BhFile
	fn := genWalker(&files)
	err := filepath.Walk(g_Model.dir, fn)
	if err != nil {
		fmt.Println("Error walking directory: ", err)
	}
	return files
}

func cleanTempFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeType == 0 && isTempName(path) {
		os.Remove(path)
	}
	return nil
}

func CleanTempFiles(dir string) {
	filepath.Walk(dir, cleanTempFile)
}
