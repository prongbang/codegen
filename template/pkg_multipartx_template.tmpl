package multipartx

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofiber/fiber/v2"
)

const (
	ContentType                  = "multipart/form-data"
	ImageTypeJPEG                = "image/jpeg"
	ImageTypePNG                 = "image/png"
	PdfType                      = "application/pdf"
	CsvType                      = "text/csv"
	XlsxTypeExcel                = "application/vnd.ms-excel"
	XlsxTypeOpenXMLWorkbook      = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	XlsxTypeOpenXMLTemplate      = "application/vnd.openxmlformats-officedocument.spreadsheetml.template"
	XlsxTypeMacroEnabledWorkbook = "application/vnd.ms-excel.sheet.macroEnabled.12"
)

// Grouped constants as slices for easier validation.
var (
	ImageTypes = []string{
		ImageTypeJPEG,
		ImageTypePNG,
	}
	PdfTypes = []string{
		PdfType,
	}
	CsvTypes = []string{
		CsvType,
	}
	XlsxTypes = []string{
		XlsxTypeExcel,
		XlsxTypeOpenXMLWorkbook,
		XlsxTypeOpenXMLTemplate,
		XlsxTypeMacroEnabledWorkbook,
	}
)

type Request interface {
	FileFields() map[string]**multipart.FileHeader
	FormFields() map[string]interface{}
}

// IsMultipartForm Helper function to check if the request is multipart form data
func IsMultipartForm(c *fiber.Ctx) bool {
	contentType := c.Get("Content-Type")
	return strings.Contains(contentType, ContentType)
}

func ValidateMimeType(header *multipart.FileHeader, allowedMimeTypes map[string]bool) error {
	file, err := header.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	// Read the file content
	buffer, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Detect MIME type
	mtype := mimetype.Detect(buffer)
	if !allowedMimeTypes[mtype.String()] {
		return fmt.Errorf("type %s unsupported", mtype.String())
	}

	return nil
}

func FileHeaderToString(fileHeader *multipart.FileHeader) ([][]string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return [][]string{}, err
	}

	// Parse the file
	r := csv.NewReader(bufio.NewReader(file))

	// Iterate through the records
	rows := [][]string{}
	for {
		// Read each record from csv
		record, e := r.Read()
		if e == io.EOF {
			break
		}

		rows = append(rows, record)
	}

	return rows, nil
}

func FileHeaderToReader(fileHeader *multipart.FileHeader) io.Reader {
	file, err := fileHeader.Open()
	if err != nil {
		return nil
	}
	return bufio.NewReader(file)
}
