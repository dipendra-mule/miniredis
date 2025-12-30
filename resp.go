package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

type Sign string // todo convert to byte later

const (
	SimpleString Sign = "+"
	Error        Sign = "-"
	Integer      Sign = ":"
	BulkString   Sign = "$"
	Array        Sign = "*"
	Null         Sign = ""
)

// ✅ Maximum allowed bulk string size (overridden by config)
var MaxBulkSize int64 = 64 * 1024 * 1024

// Maximum allowed command size (overridden by config)
var MaxCommandSize int64 = 10 * 1024 * 1024

// Maximum allowed command arguments (overridden by config)
var MaxCommandArgs int = 1000

type Resp struct {
	sign Sign
	num  int
	bulk string
	str  string
	arr  []Resp
	err  string
	// null bool
}

// *3\r\n$3\r\nSET\r\n$3\r\nKey\r\n$5\r\nValue\r\n

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

func (r *Resp) parseRespArr(reader io.Reader) error {
	rd := bufio.NewReader(reader)

	line, err := readLine(rd)
	if err != nil {
		return err
	}
	if line[0] != '*' {
		return errors.New("expcted array")
	}

	arrLen, err := strconv.Atoi(line[1:])
	if err != nil {
		return err
	}

	if arrLen > MaxCommandArgs {
		return errors.New("command exceeds maximum allowed arguments")
	}

	totalSize := int64(0)
	for i := 0; i < arrLen; i++ {
		bulk := r.parseBulkStr(rd)

		// ✅ propagate bulk parsing error upward
		if bulk.sign == Error {
			return errors.New(bulk.str)
		}

		if bulk.sign == BulkString {
			totalSize += int64(len(bulk.bulk))
			if totalSize > MaxCommandSize {
				return errors.New("command exceeds maximum allowed size")
			}
		}

		r.arr = append(r.arr, bulk)
	}

	return nil
}

func (r *Resp) parseBulkStr(reader *bufio.Reader) Resp {
	line, err := readLine(reader)
	if err != nil {
		log.Println("error in parseBulkStr():", err)
		return Resp{}
	}

	n, err := strconv.Atoi(line[1:])
	if err != nil {
		fmt.Println(err)
		return Resp{}
	}

	// ✅ enforce limit BEFORE allocation
	if int64(n) > MaxBulkSize {
		log.Println("bulk string exceeds maximum allowed size:", n)
		return Resp{
			sign: Error,
			str:  "bulk string exceeds maximum allowed size",
		}
	}

	bulkBuf := make([]byte, n+2)
	if _, err := io.ReadFull(reader, bulkBuf); err != nil {
		fmt.Println(err)
		return Resp{}
	}

	bulk := string(bulkBuf[:n])
	return Resp{
		sign: BulkString,
		bulk: bulk,
	}
}
