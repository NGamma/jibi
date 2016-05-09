package jibi

import (
	"archive/zip"
	"io/ioutil"
	"strings"
)

// BytesToWord simply converts two Byter objects into a Word.
func BytesToWord(high, low Byter) Word {
	return Word(uint16(high.Byte())<<8 + uint16(low.Byte()))
}

func readRomZipFile(filename string) ([]byte, error) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".gb") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			buf, err := ioutil.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			return buf, nil
		}
	}
	return []byte{}, nil
}

// ReadRomFile reads the file named by filename and returns the contents.
// If filename ends with ".zip" it will return the uncompressed contents in
// the first file in the archive matching the pattern "*.gb"
func ReadRomFile(filename string) ([]Byte, error) {
	var buf []byte
	var err error
	if strings.HasSuffix(filename, ".zip") {
		buf, err = readRomZipFile(filename)
	} else {
		buf, err = ioutil.ReadFile(filename)
	}
	if err != nil {
		return nil, err
	}
	r := make([]Byte, len(buf))
	for i, b := range buf {
		r[i] = Byte(b)
	}
	return r, nil
}
