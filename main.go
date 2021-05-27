package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"regexp"
)

// Regex from https://github.com/GerbenJavado/LinkFinder/blob/master/linkfinder.py

var regex1 string = "((?:[a-zA-Z]{1,10}://|//)[^\"'/]{1,}\\.[a-zA-Z]{2,}[^\"']{0,})"                                  // Basic URLS
var regex2 string = "((?:/|\\.\\./|\\./)[^\"'><,;| *()(%%$^/\\\\\\[\\]][^\"'><,;|()]{1,})"                            // Path that start with ../, / or ./
var regex3 string = "([a-zA-Z0-9_\\-/]{1,}/[a-zA-Z0-9_\\-/]{1,}\\.(?:[a-zA-Z]{1,4}|action)(?:[\\?|#][^\"|']{0,}|))"   // Relative endpoint with extension and ? or # parameters
var regex4 string = "([a-zA-Z0-9_\\-/]{1,}/[a-zA-Z0-9_\\-/]{3,}(?:[\\?|#][^\"|']{0,}|))"                              // Relative endpoints without extensions but with parameters
var regex5 string = "([a-zA-Z0-9_\\-]{1,}\\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\\?|#][^\"|']{0,}|))" // Filenames

var regex_full string = "(?:\"|')" + "(" + regex1 + "|" + regex2 + "|" + regex3 + "|" + regex4 + "|" + regex5 + ")" + "(?:\"|')"

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}
	return (err != nil)
}

func contentParser(content []byte, pattern string) [][]byte {
	re := regexp.MustCompile(pattern)
	return re.FindAll(content, -1)
}

func printResults(result [][]byte) {

	for _, element := range result {
		fmt.Println(string(element))
	}
}

func banner() {
	var banner string = "JSLoot v0.1\n"
	fmt.Println(banner)
}

func main() {

	banner()

	var js string

	flag.Parse()

	if flag.NArg() > 0 {
		js = flag.Arg(0)
	}

	content, err := ioutil.ReadFile(js)
	if isError(err) {
		return
	}

	result := contentParser(content, regex_full)
	printResults(result)
}
