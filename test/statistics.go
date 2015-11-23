package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("name | success | diff ")

	files, err := ioutil.ReadDir("/tmp/log/")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	count := 0
	successCount := 0
	totalDiff := 0

	for _, file := range files {
		count++
		name := file.Name()
		success, diff, err := statistic(path.Join("/tmp/log/", name))
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Printf("%s    %t    %d \n", name, success, diff)

		if success {
			successCount++
			totalDiff = totalDiff + diff
		}

	}

	fmt.Println("======================================================")
	fmt.Printf("total: %d \n", count)
	fmt.Printf("success: %d \n", successCount)
	d := 0
	if successCount != 0 {
		d = totalDiff / successCount
	}
	fmt.Printf("diff: %d \n", d)
}

func statistic(file string) (success bool, diff int, err error) {
	success = false
	diff = 0

	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		text := scanner.Text()
		if strings.Contains(text, "[statistics_success]") {
			success = true
			strs := strings.SplitN(text, " ", -1)
			diff, err = strconv.Atoi(strs[len(strs)-1])
			if err != nil {
				return false, 0, err
			}

		} else if strings.Contains(text, "[statistics_fail]") {
			success = false
		}
	}

	return success, diff, nil
}
