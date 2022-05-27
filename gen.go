package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	contentDir     = "content"
	outputDir      = "posts"
	postDateLayout = "Jan 2, 2006"
)

type Post struct {
	name, title string    // Always present.
	date        time.Time // Always present.
	redirectTo  string    // "" in case of a regular post.
}

func main() {
	posts := collectPosts()
	genPostPages(posts)
	genIndexHtml(posts)
}

func collectPosts() (posts []Post) {
	postNames, err := ioutil.ReadDir(contentDir)
	if err != nil {
		log.Fatalf("Cannot read dir '%s': %v.", contentDir, err)
	}

	for _, postName := range postNames {
		baseName := postName.Name()
		extension := filepath.Ext(baseName)

		var post Post
		post.name = strings.TrimSuffix(baseName, extension)
		parseMetadata(readPostContent(baseName), &post)
		posts = append(posts, post)
	}

	return posts
}

func readPostContent(postBaseName string) string {
	postFilename := fmt.Sprintf("%s/%s", contentDir, postBaseName)
	content, err := os.ReadFile(postFilename)
	if err != nil {
		log.Fatalf("Cannot read file '%s': %v.", postFilename, err)
	}

	return string(content)
}

func parseMetadata(content string, post *Post) {
	for _, line := range strings.Split(string(content), "\n") {
		title := parseMetadataField(line, "title")
		if title != "" {
			titleWithoutDoubleQuotes := title[1 : len(title)-1]
			post.title = titleWithoutDoubleQuotes
		}

		if post.redirectTo == "" {
			post.redirectTo = parseMetadataField(line, "redirect")
		}

		parsePostDate(post, line)
	}

	if post.title == "" {
		log.Fatalf("Cannot find a title in '%s'.", post.name)
	}
	if post.date.Year() == 0 {
		log.Fatalf("Cannot find a date in '%s'.", post.name)
	}
}

func parsePostDate(post *Post, line string) {
	if dateStr := parseMetadataField(line, "date"); dateStr != "" {
		date, err := time.Parse(postDateLayout, dateStr)
		if err != nil {
			log.Fatalf("Cannot parse date in '%s': %v.", post.name, err)
		}
		post.date = date
	}
}

func parseMetadataField(line, fieldName string) string {
	fieldPrefix := fmt.Sprintf("%s: ", fieldName)

	if strings.HasPrefix(line, fieldPrefix) {
		return strings.TrimPrefix(line, fieldPrefix)
	}

	return ""
}

func genPostPages(posts []Post) {
	for _, post := range posts {
		if post.redirectTo != "" {
			genRedirectHtml(&post)
		} else {
			invokePandoc(&post)
		}
	}
}

func genRedirectHtml(post *Post) {
	htmlFilename := fmt.Sprintf("%s/%s.html", outputDir, post.name)
	file, err := os.Create(htmlFilename)
	defer file.Close()
	if err != nil {
		log.Fatalf("Cannot create '%s': %v.", htmlFilename, err)
	}

	w := bufio.NewWriter(file)
	defer w.Flush()

	fmt.Fprintf(w, "<!DOCTYPE html><html><script>window.location.replace(\"%s\");</script></html>", post.redirectTo)
}

func invokePandoc(post *Post) {
	cmd := exec.Command(
		"pandoc", fmt.Sprintf("%s/%s.md", contentDir, post.name),
		"--output", fmt.Sprintf("%s/%s.html", outputDir, post.name),
		"--standalone",
		"-H", "header.html",
		"--table-of-contents",
		"--citeproc",
		"--css", "../style.css",
		"--include-after-body", "utterances.html",
		"--include-in-header", "post_header_aux.html")

	output, err := cmd.CombinedOutput()
	defer fmt.Print(string(output))
	if err != nil {
		log.Fatal(err)
	}
}

func genIndexHtml(posts []Post) {
	file, err := os.Create("index.html")
	if err != nil {
		log.Fatalf("Cannot create 'index.html': %v.", err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	defer w.Flush()
	fmt.Fprintf(w, "<!DOCTYPE html>\n<html>\n<head>\n")
	fmt.Fprintf(w, "<title>Hirrolot's Blog</title>\n")

	err = appendFile(w, "header.html")
	if err != nil {
		log.Fatalf("Cannot append 'header.html': %s.", err)
	}

	fmt.Fprintf(w, "<link rel=\"stylesheet\" href=\"style.css\" />\n")
	fmt.Fprintf(w, "<link rel=\"shortcut icon\" href=\"myself.png\" type=\"image/x-icon\">\n")
	fmt.Fprintf(w, "<script src=\"script.js\"></script>\n\n")

	fmt.Fprintf(w, "</head>\n<body>\n")
	fmt.Fprintf(w, "<h1 class=\"blog-title\">Hirrolot's Blog</h1>\n\n")
	err = appendFile(w, "badges.html")
	if err != nil {
		log.Fatalf("Cannot append 'badges.html': %s.", err)
	}

	genPostsHistory(w, posts)

	fmt.Fprintln(w, "</body>\n</html>")
}

func appendFile(w io.Writer, filename string) error {
	headerHtml, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, bytes.NewReader(headerHtml))
	if err != nil {
		return err
	}

	return nil
}

func genPostsHistory(w io.Writer, posts []Post) error {
	fmt.Fprintf(w, "<div class=\"posts-history\">\n")

	minYear, maxYear := minPostYear(posts), maxPostYear(posts)

	for year := maxYear; year >= minYear; year-- {
		for month := time.December; month >= time.January; month-- {
			for day := 31; day >= 1; day-- {
				for _, post := range posts {
					if post.date.Year() != year || post.date.Month() != month || post.date.Day() != day {
						continue
					}

					var postClass string
					if post.redirectTo != "" {
						postClass = "redirect-post"
					} else {
						postClass = "regular-post"
					}

					fmt.Fprintf(w,
						"<div class=\"post-link\"><a href=\" %s/%s.html\">%s</a><span class=\"%s\"></span><br><span class=\"post-date\">%s</span></div>\n",
						outputDir, post.name, post.title, postClass,
						post.date.Format(postDateLayout))
				}
			}
		}
	}

	fmt.Fprintln(w, "</div>") // class="posts-history"
	return nil
}

func minPostYear(posts []Post) int {
	result := 9999
	for _, post := range posts {
		if result > post.date.Year() {
			result = post.date.Year()
		}
	}

	return result
}

func maxPostYear(posts []Post) int {
	result := 0
	for _, post := range posts {
		if result < post.date.Year() {
			result = post.date.Year()
		}
	}

	return result
}
