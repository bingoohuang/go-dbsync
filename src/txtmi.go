package main

import (
	"flag"
	"fmt"
	"github.com/dgiagio/getpass"
	"io/ioutil"
	"os"
	"regexp"
	"./myutil"
)

/*
对文本中形如${?:clear text}的文本进行加密，加密后，显示为${AES:密文}，反之解密
*/

func main() {
	mode := flag.String("mode", "encode", "mode(encode/decode/src)")
	key := flag.String("key", "", "key to encyption or decyption")
	infile := flag.String("infile", "", "input file")
	outfile := flag.String("outfile", "", "output file")
	flag.Parse()
	if *infile == "" {
		msg := "file argument is required!\n"
		printErrorAndExit(msg)
	}

	keyStr := *key
	if *key == "" {
		keyStr, _ = getpass.GetPassword("Please input the key: ")
	}
	keyStr = myutil.FixStrLength(keyStr, 16)

	txtBytes, err := ioutil.ReadFile(*infile)
	checkError(err)

	txt := string(txtBytes)

	var regex *regexp.Regexp
	var replaceFunc func(groups []string) string

	switch *mode {
	default:
		printErrorAndExit("mode argument should be ecode/decode/src!\n")

	case "encode":
		regex, _ = regexp.Compile("\\$\\{\\?:(.*?)\\}")
		replaceFunc = func(groups []string) string {
			cipher, _ := myutil.CBCEncrypt(keyStr, groups[1])
			return "${AES:" + cipher + "}"
		}
	case "decode":
		regex, _ = regexp.Compile("\\$\\{AES:(.*?)\\}")
		replaceFunc = func(groups []string) string {
			clear, _ := myutil.CBCDecrypt(keyStr, groups[1])
			return clear
		}
	case "src":
		regex, _ = regexp.Compile("\\$\\{AES:(.*?)\\}")
		replaceFunc = func(groups []string) string {
			clear, _ := myutil.CBCDecrypt(keyStr, groups[1])
			return "${?:" + clear + "}"
		}
	}

	result := myutil.ReplaceAllGroupFunc(regex, txt, replaceFunc)
	WriteOutput(*outfile, result)
}

func printErrorAndExit(msg string) {
	fmt.Printf(msg)
	Usage()
	os.Exit(-1)
}

func WriteOutput(outfile, result string) {
	if outfile == "" {
		fmt.Printf("%s\n", result)
	} else {
		err := ioutil.WriteFile(outfile, []byte(result), 0644)
		checkError(err)
	}
}
func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}


