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

// OUTPUT COLORS

var colorReset string = "\033[0m"
var colorRed string = "\033[31m"
var colorGreen string = "\033[32m"
var colorYellow string = "\033[33m"
var colorBlue string = "\033[34m"
var colorPurple string = "\033[35m"
var colorCyan string = "\033[36m"
var colorWhite string = "\033[37m"

// REGEX PATTERNS
//
// URL Regex from https://github.com/GerbenJavado/LinkFinder/blob/master/linkfinder.py

var url_regex1 string = "((?:[a-zA-Z]{1,10}://|//)[^\"'/]{1,}\\.[a-zA-Z]{2,}[^\"']{0,})"                                  // Basic URLS
var url_regex2 string = "((?:/|\\.\\./|\\./)[^\"'><,;| *()(%%$^/\\\\\\[\\]][^\"'><,;|()]{1,})"                            // Path that start with ../, / or ./
var url_regex3 string = "([a-zA-Z0-9_\\-/]{1,}/[a-zA-Z0-9_\\-/]{1,}\\.(?:[a-zA-Z]{1,4}|action)(?:[\\?|#][^\"|']{0,}|))"   // Relative endpoint with extension and ? or # parameters
var url_regex4 string = "([a-zA-Z0-9_\\-/]{1,}/[a-zA-Z0-9_\\-/]{3,}(?:[\\?|#][^\"|']{0,}|))"                              // Relative endpoints without extensions but with parameters
var url_regex5 string = "([a-zA-Z0-9_\\-]{1,}\\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\\?|#][^\"|']{0,}|))" // Filenames
var url_regex_full string = url_regex1 + "|" + url_regex2 + "|" + url_regex3 + "|" + url_regex4 + "|" + url_regex5
var aws_keys_regex string = "(AKIA|A3T|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{12,}"
var b64_regex string = "(eyJ|YTo|Tzo|PD[89]|aHR0cHM6L|aHR0cDo|rO0)[%a-zA-Z0-9+/]+={0,2}"
var ip_regex string = "(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])"
var quote_regex string = "(?:\"|')"

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}
	return (err != nil)
}

func contentParser(content []byte, pattern string) [][]byte {
	re, err := regexp.Compile(pattern)
	if isError(err) {
		os.Exit(1)
	}
	return re.FindAll(content, -1)
}

func testMatch(regex string, target []byte) bool {
	re := regexp.MustCompile(regex)
	return re.Match(target)
}

func colorString(color string, output string) string {
	return "[" + color + output + colorReset + "] "
}

func showMatchType(target []byte) (output string) {
	if testMatch(url_regex1, target) {
		output = colorString(colorCyan, "URL")
	} else if testMatch(url_regex5, target) {
		output = colorString(colorCyan, "File")
	} else if testMatch(url_regex2, target) {
		output = colorString(colorCyan, "Path")
	} else if testMatch(url_regex3, target) {
		output = colorString(colorCyan, "Endpoint")
	} else if testMatch(url_regex4, target) {
		output = colorString(colorCyan, "Endpoint")
	} else if testMatch(aws_keys_regex, target) {
		output = colorString(colorYellow, "AWS-Key")
	} else if testMatch(b64_regex, target) {
		output = colorString(colorBlue, "Base64")
	} else if testMatch(ip_regex, target) {
		output = colorString(colorGreen, "IPv4")
	} else {
		output = colorString(colorPurple, "Custom")
	}
	return
}

func printResults(filename string, result [][]byte, verbose bool) {
	for _, element := range result {
		if verbose {
			fmt.Print(showMatchType(element))
		}
		if filename != "" {
			fmt.Println(filename + ":\t" + string(element))
		} else {
			fmt.Println(string(element))
		}

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

func buildRegex(grep string, custom_expression string) (regex string) {
	if custom_expression != "" {
		regex += custom_expression
	}

	if strings.Contains(grep, "5") {
		return
	} else if grep == "" {
		grep = "1,2,3,4"
	}

	g := strings.Split(grep, ",")

	if len(g) > 0 {
		if custom_expression != "" {
			regex += "|"
		}

		regex += quote_regex + "("
		for index, element := range g {
			switch string(element) {
			case "1":
				regex += url_regex_full
			case "2":
				regex += aws_keys_regex
			case "3":
				regex += b64_regex
			case "4":
				regex += ip_regex
			default:
				fmt.Printf("No such grep case: %s\n", string(element))
				os.Exit(1)
			}
			if index+1 < len(g) {
				regex += "|"
			}
		}
		regex += ")" + quote_regex
	}

	return
}

func lootJS(content []byte, regex string, filename string, show_filename, verbose bool) {
	result := contentParser(content, regex)
	if !show_filename {
		filename = ""
	}
	printResults(filename, result, verbose)
}

func lootJSOnURL(url string, regex string, show_matching_location bool, verbose bool) {
	var content []byte
	response := getResponseFromURL(url)
	if isResponseJavaScript(response) {
		content = getContentFromResponse(response)
		lootJS(content, regex, url, show_matching_location, verbose)
	} else if isResponseHTML(response) {
		urls := getJsURLsFromHTML(response)
		for _, url := range urls {
			response = getResponseFromURL(url)
			content = getContentFromResponse(response)
			lootJS(content, regex, url, show_matching_location, verbose)
		}
	} else {
		return
	}
}

func lootJSOnFile(file_list []string, regex string, show_matching_location bool, verbose bool) {
	var content []byte
	for _, file := range file_list {
		content = getContentFromFile(file)
		lootJS(content, regex, file, show_matching_location, verbose)
	}
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
		"JSLoot by famos0",
		"",
		"Looting URLs, IPv4 addresses, base64 encoded stuff and aws-keys from JavaScript",
		"",
		"-- WHERE TO LOOT ? -- ",
		" -u, --url <url>\t\tLoot from on the URL",
		" -f, --file <path>\t\tLoot from a local file",
		" -d, --directory <path>\t\tLoot from a directory",
		" -r, --recurse <path>\t\tTo combine with -d option. Loot recursively",
		" -s, --stdin\t\t\tLoot from URLs given by Stdin",
		"",
		"-- WHAT TO LOOT ? -- ",
		" -e, --regexp <PATTERNS>\tLoot with a custom pattern",
		" -g, --grep-pattern <1,2,...>\tWhen specified, custom the looting patterns :",
		"\t\t\t\t- 1 : Looting URLs",
		"\t\t\t\t- 2 : Looting AWS Keys",
		"\t\t\t\t- 3 : Looting Base64 artifacts",
		"\t\t\t\t- 4 : Looting IPv4 addresses",
		"\t\t\t\t- 5 : Looting nothing on the default patterns",
		"\t\t\t\t      Can be used in complement with -e",
		"",
		"-- SHOW THE LOOT -- ",
		" -H, --with-filename\t\tShow filename/URL of loot location",
		" -v, --verbose\t\t\tPrint with detailed output and colors",
	}

	fmt.Println(strings.Join(helper, "\n"))
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
	flag.StringVar(&dir, "directory", "", "")
	flag.StringVar(&dir, "d", "", "")

	var recurse bool
	flag.BoolVar(&recurse, "recurse", false, "")
	flag.BoolVar(&recurse, "r", false, "")

	var stdin bool
	flag.BoolVar(&stdin, "stdin", false, "")
	flag.BoolVar(&stdin, "s", false, "")

	var grep string
	flag.StringVar(&grep, "grep-pattern", "", "")
	flag.StringVar(&grep, "g", "", "")

	var custom_regex string
	flag.StringVar(&custom_regex, "regexp", "", "")
	flag.StringVar(&custom_regex, "e", "", "")

	var show_matching_location bool
	flag.BoolVar(&show_matching_location, "with-filename", false, "")
	flag.BoolVar(&show_matching_location, "H", false, "")

	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "")
	flag.BoolVar(&verbose, "v", false, "")

	flag.Parse()

	regex := buildRegex(grep, custom_regex)

	if regex != "" {
		if url != "" {
			lootJSOnURL(url, regex, show_matching_location, verbose)

		} else if file != "" {
			lootJSOnFile([]string{file}, regex, show_matching_location, verbose)
		} else if dir != "" {
			if !isDir(dir) {
				return
			}
			var file_list []string
			if recurse {
				file_list = listFilesInDirRec(dir)
			} else {
				file_list = listFilesInDir(dir)
			}

			lootJSOnFile(file_list, regex, show_matching_location, verbose)
		} else if stdin {
			sc := bufio.NewScanner(os.Stdin)
			for sc.Scan() {
				url = sc.Text()
				lootJSOnURL(url, regex, show_matching_location, verbose)
			}
		} else {
			showHelper()
			return
		}
	}
}
