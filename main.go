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

var url_regex1 string = "((?:[a-zA-Z]{1,10}://|//)[^\"'/]{1,}\\.[a-zA-Z]{2,}[^\"']{0,})"                                  // Basic URLS
var url_regex2 string = "((?:/|\\.\\./|\\./)[^\"'><,;| *()(%%$^/\\\\\\[\\]][^\"'><,;|()]{1,})"                            // Path that start with ../, / or ./
var url_regex3 string = "([a-zA-Z0-9_\\-/]{1,}/[a-zA-Z0-9_\\-/]{1,}\\.(?:[a-zA-Z]{1,4}|action)(?:[\\?|#][^\"|']{0,}|))"   // Relative endpoint with extension and ? or # parameters
var url_regex4 string = "([a-zA-Z0-9_\\-/]{1,}/[a-zA-Z0-9_\\-/]{3,}(?:[\\?|#][^\"|']{0,}|))"                              // Relative endpoints without extensions but with parameters
var url_regex5 string = "([a-zA-Z0-9_\\-]{1,}\\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\\?|#][^\"|']{0,}|))" // Filenames
var aws_keys_regex string = "(AKIA|A3T|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{12,}"
var b64_regex string = "(eyJ|YTo|Tzo|PD[89]|aHR0cHM6L|aHR0cDo|rO0)[%a-zA-Z0-9+/]+={0,2}"
var ip_regex string = "(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])"

var regex_full string = "(?:\"|')" + "(" + url_regex1 + "|" + url_regex2 + "|" + url_regex3 + "|" + url_regex4 + "|" + url_regex5 + "|" + aws_keys_regex + "|" + b64_regex + "|" + ip_regex + ")" + "(?:\"|')"

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

func listFilesInDirRec(dirname string) (files_in_dir []string) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	for _, file := range files {
		if isDir(file.Name()) {
			for _, filerec := range listFilesInDirRec(file.Name()) {
				files_in_dir = append(files_in_dir, filerec)
			}
		} else {
			files_in_dir = append(files_in_dir, file.Name())
		}
	}
	return
}

func listFilesInDir(dirname string) (files_in_dir []string) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	for _, file := range files {
		if !isDir(file.Name()) {
			files_in_dir = append(files_in_dir, file.Name())
		}
	}
	return
}

func isDir(dir string) bool {
	fi, err := os.Stat(dir)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return fi.IsDir()
}

func showHelper() {
	helper := []string{
		"JSLoot",
		"",
		"Looting URLs, IPv4 addresses, base64 encoded stuff and aws-keys from JavaScript",
		"",
		" -u, --url <url>		Loot from on the URL",
		" -f, --file <path>		Loot from a local file",
		" -d, --dir <path>		Loot from a directory but no recursive",
		" -D, --Dir <path>		Loot from a directory recursively",
		" -s, --stdin			Loot from URLs given by Stdin",
		"",
	}

	fmt.Fprintf(os.Stderr, strings.Join(helper, "\n"))
}

func init() {
	flag.Usage = func() {
		showHelper()
	}
}

func main() {

	var url string
	flag.StringVar(&url, "url", "", "")
	flag.StringVar(&url, "u", "", "")

	var file string
	flag.StringVar(&file, "file", "", "")
	flag.StringVar(&file, "f", "", "")

	var dir string
	flag.StringVar(&dir, "dir", "", "")
	flag.StringVar(&dir, "d", "", "")

	var directory_rec string
	flag.StringVar(&directory_rec, "Dir", "", "")
	flag.StringVar(&directory_rec, "D", "", "")

	var stdin bool
	flag.BoolVar(&stdin, "stdin", false, "")
	flag.BoolVar(&stdin, "s", false, "")

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

	} else if dir != "" {
		if !isDir(dir) {
			return
		}
		file_list := listFilesInDir(dir)
		for _, file := range file_list {
			content = getContentFromFile(file)
			lootJS(content)
		}

	} else if directory_rec != "" {
		if !isDir(directory_rec) {
			return
		}
		file_list := listFilesInDirRec(directory_rec)
		for _, file := range file_list {
			content = getContentFromFile(file)
			lootJS(content)
		}

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
	} else {
		showHelper()
		return
	}
}
