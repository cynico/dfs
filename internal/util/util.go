package util

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/h2non/filetype"
)

type NoSuchNodeExistsError struct {
	UUID string
}

func (e *NoSuchNodeExistsError) Error() string {
	return fmt.Sprintf("no such node exists with UUID=%s", e.UUID)
}

func CheckFile(filePath string) (int64, error) {

	fs, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("error getting file info: %s", err)
	}

	if fs.IsDir() {
		return 0, fmt.Errorf("please provide a file, not a directory")
	}

	// Checking the file type by reading its magic number
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return 0, err
	}

	kind, _ := filetype.Match(buffer)
	if kind.MIME.Value != "video/mp4" {
		return 0, fmt.Errorf("only mp4 video files are allowed")
	}

	return fs.Size(), nil
}

func SendFile(filePath string, fileSize int64, c net.Conn) error {

	// Send the file size first
	if err := binary.Write(c, binary.LittleEndian, fileSize); err != nil {
		return err
	}

	// Then open the file and send it
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file %s: %s", filePath, err.Error())
	}

	n, err := io.CopyN(c, file, int64(fileSize))
	if err != nil {
		return fmt.Errorf("error sending file %s: %s", filePath, err.Error())
	}
	if n != int64(fileSize) {
		return fmt.Errorf("partial write. sent %d bytes out of %d", n, fileSize)
	}

	return nil
}

func ReceiveFile(filePath string, c net.Conn) (int64, error) {

	// Receive the file size first
	var fileSize int64
	if err := binary.Read(c, binary.LittleEndian, &fileSize); err != nil {
		return 0, err
	}

	// Creating the file on disk
	file, err := os.Create(filePath)
	if err != nil {
		os.Remove(filePath)
		return 0, fmt.Errorf("failed to open file for writing at %s: %s", filePath, err.Error())
	}

	// Reading file data from connection into the file
	n, err := io.CopyN(file, c, fileSize)
	if err != nil {
		os.Remove(filePath)
		return 0, fmt.Errorf("error reading from connection into file: %s", err.Error())
	} else if n != fileSize {
		os.Remove(filePath)
		return 0, fmt.Errorf("partial read from connection. read %d out of %d", n, fileSize)
	}

	return fileSize, nil
}

func SendString(str string, c net.Conn) error {

	buf := []byte(str)
	if err := binary.Write(c, binary.LittleEndian, int64(len(buf))); err != nil {
		return err
	}

	if err := binary.Write(c, binary.LittleEndian, []byte(buf)); err != nil {
		return err
	}

	return nil
}

func ReceiveString(c net.Conn) (string, error) {

	var stringSize int64
	if err := binary.Read(c, binary.LittleEndian, &stringSize); err != nil {
		return "", err
	}

	// Read file name
	buf := make([]byte, stringSize)
	if err := binary.Read(c, binary.LittleEndian, &buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

func CountTrue(bools ...bool) int {
	count := 0
	for _, b := range bools {
		if b {
			count++
		}
	}
	return count
}

func RemoveElementFromSlice[T any](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}
