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

type Resp struct {
	sign Sign
	num  int
	bulk string
	str  string
	arr  []Resp
	err  string
	// null    bool
}

// *3\r\n$3\r\nSET\r\n$3\r\nKey\r\n$5\r\nValue\r\n

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

func (r *Resp) parseRespArr(rd *bufio.Reader) error {

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

	for range arrLen {
		bulk, err := r.parseBulkStr(rd)
		if err != nil {
			log.Println("error in parseRespArr():", err)
			return err
		}
		r.arr = append(r.arr, bulk)
	}
	return nil
}

func (r *Resp) parseBulkStr(reader *bufio.Reader) (Resp, error) {
	line, err := readLine(reader)
	if err != nil {
		log.Println("error in parseBulkStr():", err)
		return Resp{}, err
	}

	n, err := strconv.Atoi(line[1:])
	if err != nil {
		fmt.Println(err)
		return Resp{}, err
	}

	bulkBuf := make([]byte, n+2)
	if _, err := io.ReadFull(reader, bulkBuf); err != nil {
		fmt.Println(err)
		return Resp{}, err
	}

	bulk := string(bulkBuf[:n])
	return Resp{
		sign: BulkString,
		bulk: bulk,
	}, nil
}
