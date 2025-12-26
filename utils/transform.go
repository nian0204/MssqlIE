package utils

import (
	"io"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func GetTransformersWrite(writer io.Writer, format string) io.Writer {
	switch strings.ToLower(format) {
	case "gbk":
		return transform.NewWriter(writer, simplifiedchinese.GBK.NewEncoder())
	case "iso-8859-1":
		return transform.NewWriter(writer, charmap.ISO8859_1.NewEncoder())
	case "iso-8859-2":
		return transform.NewWriter(writer, charmap.ISO8859_2.NewEncoder())
	case "iso-8859-3":
		return transform.NewWriter(writer, charmap.ISO8859_3.NewEncoder())
	case "iso-8859-4":
		return transform.NewWriter(writer, charmap.ISO8859_4.NewEncoder())
	case "iso-8859-9":
		return transform.NewWriter(writer, charmap.ISO8859_9.NewEncoder())
	case "iso-8859-10":
		return transform.NewWriter(writer, charmap.ISO8859_10.NewEncoder())
	case "iso-8859-13":
		return transform.NewWriter(writer, charmap.ISO8859_13.NewEncoder())
	case "iso-8859-14":
		return transform.NewWriter(writer, charmap.ISO8859_14.NewEncoder())
	case "iso-8859-15":
		return transform.NewWriter(writer, charmap.ISO8859_15.NewEncoder())
	case "iso-8859-16":
		return transform.NewWriter(writer, charmap.ISO8859_16.NewEncoder())
	case "cp1252", "windows-1252":
		return transform.NewWriter(writer, charmap.Windows1252.NewEncoder())
	default: // raw
		return writer
	}
}

func GetTransformersRead(reader io.Reader, format string) io.Reader {
	switch strings.ToLower(format) {
	case "gbk":
		return transform.NewReader(reader, simplifiedchinese.GBK.NewDecoder())
	case "iso-8859-1":
		return transform.NewReader(reader, charmap.ISO8859_1.NewDecoder())
	case "iso-8859-2":
		return transform.NewReader(reader, charmap.ISO8859_2.NewDecoder())
	case "iso-8859-3":
		return transform.NewReader(reader, charmap.ISO8859_3.NewDecoder())
	case "iso-8859-4":
		return transform.NewReader(reader, charmap.ISO8859_4.NewDecoder())
	case "iso-8859-9":
		return transform.NewReader(reader, charmap.ISO8859_9.NewDecoder())
	case "iso-8859-10":
		return transform.NewReader(reader, charmap.ISO8859_10.NewDecoder())
	case "iso-8859-13":
		return transform.NewReader(reader, charmap.ISO8859_13.NewDecoder())
	case "iso-8859-14":
		return transform.NewReader(reader, charmap.ISO8859_14.NewDecoder())
	case "iso-8859-15":
		return transform.NewReader(reader, charmap.ISO8859_15.NewDecoder())
	case "iso-8859-16":
		return transform.NewReader(reader, charmap.ISO8859_16.NewDecoder())
	case "cp1252", "windows-1252":
		return transform.NewReader(reader, charmap.Windows1252.NewDecoder())
	default: // raw
		return reader
	}
}
