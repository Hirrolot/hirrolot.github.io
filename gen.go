package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	contentDir     = "content"
	outputDir      = "posts"
	postDateLayout = "Jan 2, 2006"
)

type Post struct {
	Name, Title string    // Always present.
	Date        time.Time // Always present.
	RedirectTo  string    // "" in case of a regular post.
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
		post.Name = strings.TrimSuffix(baseName, extension)
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
			post.Title = titleWithoutDoubleQuotes
		}

		if post.RedirectTo == "" {
			post.RedirectTo = parseMetadataField(line, "redirect")
		}

		parsePostDate(post, line)
	}

	checkPostMetadata(post)
}

func parsePostDate(post *Post, line string) {
	if dateStr := parseMetadataField(line, "date"); dateStr != "" {
		date, err := time.Parse(postDateLayout, dateStr)
		if err != nil {
			log.Fatalf("Cannot parse date in '%s': %v.", post.Name, err)
		}
		post.Date = date
	}
}

func parseMetadataField(line, fieldName string) string {
	fieldPrefix := fmt.Sprintf("%s: ", fieldName)

	if strings.HasPrefix(line, fieldPrefix) {
		return strings.TrimPrefix(line, fieldPrefix)
	}

	return ""
}

func checkPostMetadata(post *Post) {
	if post.Title == "" {
		log.Fatalf("Cannot find a title in '%s'.", post.Name)
	}
	if post.Date.Year() == 0 {
		log.Fatalf("Cannot find a date in '%s'.", post.Name)
	}
}

func genPostPages(posts []Post) {
	for _, post := range posts {
		if post.RedirectTo != "" {
			genRedirectHtml(&post)
		} else {
			invokePandoc(&post)
		}
	}
}

func genRedirectHtml(post *Post) {
	htmlFilename := fmt.Sprintf("%s/%s.html", outputDir, post.Name)
	file, err := os.Create(htmlFilename)
	defer file.Close()
	if err != nil {
		log.Fatalf("Cannot create '%s': %v.", htmlFilename, err)
	}

	w := bufio.NewWriter(file)
	defer w.Flush()

	t := template.Must(template.ParseFiles("templates/redirect.tmpl"))

	err = t.ExecuteTemplate(w, "redirect.tmpl", post)
	if err != nil {
		log.Fatal(err)
	}
}

func invokePandoc(post *Post) {
	cmd := exec.Command(
		"pandoc", fmt.Sprintf("%s/%s.md", contentDir, post.Name),
		"--output", fmt.Sprintf("%s/%s.html", outputDir, post.Name),
		"--standalone",
		"-H", "header.html",
		"--table-of-contents",
		"--citeproc",
		"--css", "../style.css",
		"--include-after-body", "utterances.html",
		"--include-in-header", "post-header.html")

	output, err := cmd.CombinedOutput()
	defer fmt.Print(string(output))
	if err != nil {
		log.Fatal(err)
	}
}

type IndexTemplate struct {
	OutputDir string
	Posts     []Post
	Contacts  []Contact
}

type Contact struct {
	Link, Img, Description string
}

func genIndexHtml(posts []Post) {
	file, err := os.Create("index.html")
	if err != nil {
		log.Fatalf("Cannot create 'index.html': %v.", err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	defer w.Flush()

	t := template.Must(template.ParseFiles("templates/index.tmpl", "header.html"))

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	err = t.ExecuteTemplate(w, "index.tmpl",
		IndexTemplate{OutputDir: outputDir, Posts: posts, Contacts: readContacts()})
	if err != nil {
		log.Fatal(err)
	}
}

func readContacts() []Contact {
	contactsData, err := os.ReadFile("contacts.json")
	if err != nil {
		log.Fatalf("Cannot read 'contacts.json': %v.", err)
	}

	var contacts []Contact
	if err := json.Unmarshal(contactsData, &contacts); err != nil {
		log.Fatal(err)
	}

	return contacts
}
