package appdist

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func writeDebArchive(output, controlTar, dataTar string, epoch time.Time) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	if _, err := file.WriteString("!<arch>\n"); err != nil {
		_ = file.Close()
		return err
	}
	if err := writeArMember(file, "debian-binary", []byte("2.0\n"), epoch); err != nil {
		_ = file.Close()
		return err
	}
	for _, item := range []struct{ name, path string }{{"control.tar.gz", controlTar}, {"data.tar.gz", dataTar}} {
		data, err := os.ReadFile(item.path)
		if err != nil {
			_ = file.Close()
			return err
		}
		if err := writeArMember(file, item.name, data, epoch); err != nil {
			_ = file.Close()
			return err
		}
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Chtimes(output, epoch, epoch)
}

func writeArMember(writer io.Writer, name string, data []byte, epoch time.Time) error {
	if len(name) > 15 {
		return fmt.Errorf("ar member name %q is too long", name)
	}
	header := fmt.Sprintf("%-16s%-12s%-6s%-6s%-8s%-10s`\n", name+"/", strconv.FormatInt(epoch.Unix(), 10), "0", "0", "100644", strconv.Itoa(len(data)))
	if len(header) != 60 {
		return fmt.Errorf("invalid ar header length for %s", name)
	}
	if _, err := io.WriteString(writer, header); err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	if len(data)%2 != 0 {
		_, err := writer.Write([]byte{'\n'})
		return err
	}
	return nil
}
