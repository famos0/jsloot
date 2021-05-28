package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
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

func checkReponseContentType(response *http.Response, content_type string) bool {
	content_type_in_response := response.Header.Get("content-type")
	return strings.Contains(content_type_in_response, content_type)
}

func isResponseJavaScript(response *http.Response) bool {
	return checkReponseContentType(response, "javascript")
}

func isResponseHTML(response *http.Response) bool {
	return checkReponseContentType(response, "html")
}

func getSrcFromTags(tokenizer html.Token) (src string) {
	for _, attribute := range tokenizer.Attr {
		if attribute.Key == "src" {
			src = attribute.Val
		}
	}
	return
}

func getJsURLsFromHTML(response *http.Response) []string {

	var urls []string
	tokenizer := html.NewTokenizer(response.Body)
	for {
		tt := tokenizer.Next()
		t := tokenizer.Token()
		err := tokenizer.Err()

		if err == io.EOF {
			break
		}

		switch tt {
		case html.ErrorToken:
			continue
		case html.StartTagToken:
			isAnchor := t.Data == "script"
			if !isAnchor {
				continue
			}

			url := getSrcFromTags(t)
			if url == "" {
				continue
			}

			hasProto := strings.Index(url, "http") == 0
			if hasProto {
				urls = append(urls, url)
			}
		}
	}
	return urls
}

func getResponseFromURL(url string) *http.Response {
	response, err := http.Get(url)
	if err != nil {
		print(err)
		return nil
	}
	return response
}

func getContentFromResponse(response *http.Response) []byte {
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		print(err)
		return nil
	}

	return body
}

func getContentFromFile(file string) []byte {
	content, err := ioutil.ReadFile(file)
	if isError(err) {
		return nil
	}

	return content
}

func lootJS(content []byte) {
	result := contentParser(content, regex_full)
	printResults(result)
}

func main() {

	var url string
	flag.StringVar(&url, "url", "", "")
	flag.StringVar(&url, "u", "", "")

	var file string
	flag.StringVar(&file, "file", "", "")
	flag.StringVar(&file, "f", "", "")

	var stdin bool
	flag.BoolVar(&stdin, "stdin", false, "")
	flag.BoolVar(&stdin, "s", false, "")

	banner()

	var content []byte

	flag.Parse()

	if url != "" {
		response := getResponseFromURL(url)
		if isResponseJavaScript(response) {
			content = getContentFromResponse(response)
			lootJS(content)
		} else if isResponseHTML(response) {
			urls := getJsURLsFromHTML(response)
			for _, url := range urls {
				response = getResponseFromURL(url)
				content = getContentFromResponse(response)
				lootJS(content)
			}
		} else {
			return
		}

	} else if file != "" {
		content = getContentFromFile(file)
		lootJS(content)
	} else if stdin {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			url = sc.Text()
			response := getResponseFromURL(url)
			if isResponseJavaScript(response) {
				content = getContentFromResponse(response)
				lootJS(content)
			} else if isResponseHTML(response) {
				urls := getJsURLsFromHTML(response)
				for _, url := range urls {
					response = getResponseFromURL(url)
					content = getContentFromResponse(response)
					lootJS(content)
				}
			} else {
				return
			}
		}
	}
}
