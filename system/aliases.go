package system

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func readFile(file *os.File) ([]byte, error) {
	buf := make([]byte, 1024)
	var output bytes.Buffer

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return []byte{}, err
		}
		output.Write(buf[:n])
	}

	return output.Bytes(), nil
}

func getConfigFile() *string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(
			fmt.Sprintf(
				"ERROR: couldn't get home dir.\nERROR MESSAGE: %s",
				err.Error(),
			),
		)
	}

	shell, err := GetUserShell()
	if err != nil {
		log.Fatalln(err.Error())
	}

	var file *os.File

	if shell == "fish" {
		file, err = os.Open(home + "/.config/fish/config.fish")
	} else if shell == "bash" {
		file, err = os.Open(home + "/.bashrc")
	} else if shell == "zsh" {
		file, err = os.Open(home + "/.zshrc")
	}

	if err != nil {
		log.Fatalln(
			fmt.Sprintf(
				"ERROR: couldn't read config file content.\nERROR MESSAGE: %s",
				err.Error(),
			),
		)
	}
	defer file.Close()

	content, err := readFile(file)
	if err != nil {
		log.Fatalln(
			fmt.Sprintf(
				"ERROR: couldn't read config file content.\nERROR MESSAGE: %s",
				err.Error(),
			),
		)
	}
	str := string(content)
	clear(content)
	return &str
}

func GetAliases() ([]string, error) {
	fileContent := getConfigFile()

	lines := strings.Split(*fileContent, "\n")
	var aliases []string

	for _, line := range lines {
		if line == "" || line[0] != 'a' {
			continue
		}
		if line[:5] == "alias" {
			aliases = append(aliases, line)
		}
	}

	if len(aliases) == 0 {
		return aliases, errors.New("no aliases found")
	}

	return aliases, nil
}
