package core

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

type Scanner struct {
	reader  *bufio.Reader
	LastTag Tag
	err     error
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{
		reader: bufio.NewReader(r),
	}
}

func (s *Scanner) Next() bool {
	// 1. 读取 Code 行
	codeLine, err := s.reader.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			s.err = err
		}
		return false
	}

	codeStr := strings.TrimSpace(codeLine)
	if codeStr == "" { // 跳过空行
		return s.Next()
	}

	code, err := strconv.Atoi(codeStr)
	if err != nil {
		s.err = err
		return false
	}

	// 2. 读取 Value 行
	valueLine, err := s.reader.ReadString('\n')
	if err != nil {
		// Value 行如果 EOF 也是不完整的
		s.err = err
		return false
	}

	// 去掉行尾的换行符，但保留 Value 开头的空格（DXF 规范要求）
	value := strings.TrimRight(valueLine, "\r\n")

	s.LastTag = Tag{Code: code, Value: value}
	return true
}

func (s *Scanner) Err() error {
	return s.err
}
