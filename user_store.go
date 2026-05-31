package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func validateUserID(userID string) error {
	if strings.TrimSpace(userID) != userID || userID == "" || len(userID) > 50 {
		return fmt.Errorf("用户ID长度需为 1-50，且不能包含首尾空格")
	}
	for _, r := range userID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("用户ID只能包含字母、数字、下划线和短横线")
	}
	return nil
}

func getUserDataPath(userID, fileName string) string {
	if err := validateUserID(userID); err != nil {
		return ""
	}
	return filepath.Join(userDataBaseDir, userID, filepath.Base(fileName))
}

func ensureUserDir(userID string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(userDataBaseDir, userID), 0755)
}

func loadUserJSONData(userID, fileName string, target interface{}) error {
	if err := ensureUserDir(userID); err != nil {
		return err
	}
	filePath := getUserDataPath(userID, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		initEmptyUserData(filePath, target)
		return nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取用户文件 %s 失败: %w", filePath, err)
	}
	if len(data) == 0 {
		initEmptyUserData(filePath, target)
		return nil
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	return json.Unmarshal(data, target)
}

func initEmptyUserData(filePath string, target interface{}) {
	switch v := target.(type) {
	case *[]UserIncorrectQuestion:
		*v = []UserIncorrectQuestion{}
	case *map[string]UserQuestionStat:
		*v = make(map[string]UserQuestionStat)
	default:
		log.Printf("加载用户数据: 文件 %s 不存在或为空，目标类型 %T 保持默认值", filePath, target)
	}
}

func saveUserJSONData(userID, fileName string, data interface{}) error {
	if err := ensureUserDir(userID); err != nil {
		return err
	}
	filePath := getUserDataPath(userID, fileName)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON序列化用户数据到 %s 失败: %w", filePath, err)
	}
	return os.WriteFile(filePath, jsonData, 0644)
}
