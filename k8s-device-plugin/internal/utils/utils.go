// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ghodss/yaml"

	"enflame/k8s-device-plugin/internal/logger"
)

func FileIsExist(file string) bool {
	_, err := os.Stat(file)
	return err == nil || os.IsExist(err)
}

func ConvertToString(content interface{}) string {
	byteVal, err := json.Marshal(content)
	if err != nil {
		logger.Warn("marshal content: %v to string failed: %v", content, err)
		return fmt.Sprintf("%v", content)
	}
	return string(byteVal)
}

type IntList []int

func (rer IntList) ContainValue(value int) bool {
	for _, intVal := range rer {
		if intVal == value {
			return true
		}
	}
	return false
}

func (rer IntList) ContainSubList(subList []int) bool {
	for _, subInt := range subList {
		if !rer.ContainValue(subInt) {
			return false
		}
	}
	return true
}

type StrList []string

func (rer StrList) ContainValue(value string) bool {
	for _, intVal := range rer {
		if intVal == value {
			return true
		}
	}
	return false
}

func ReadYamlToJson(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error(err, "read file: %s failed", filePath)
		return nil, err
	}
	jsonContent, err := yaml.YAMLToJSON(content)
	if err != nil {
		logger.Error(err, "convert file content to json failed, content: %s", string(content))
		return nil, err
	}
	return jsonContent, nil
}

func ExecCommand(name string, args ...string) error {
	// catch standard output of error
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		logger.Error(err, "execute command: '%s %s' failed, detail: %s", name, strings.Join(args, " "), stderrBuf.String())
		return fmt.Errorf(stderrBuf.String())
	}
	return nil
}
