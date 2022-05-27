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

const contentDir = "content"
const outputDir = "posts"

func main() {
	postNames, err := collectPostNames()
	if err != nil {
		log.Fatal(err)
	}

	for _, postName := range postNames {
		if err := invokePandoc(postName); err != nil {
			log.Fatal(err)
		}
	}

	err = genIndexHtml(postNames)
	if err != nil {
		log.Fatal(err)
	}
}

func genIndexHtml(postNames []string) error {
	file, err := os.Create("index.html")
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	defer w.Flush()
	fmt.Fprintf(w, "<!DOCTYPE html>\n<html>\n<head>\n")
	fmt.Fprintf(w, "<title>Hirrolot's Blog</title>\n")

	err = appendFile(w, "header.html")
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "<link rel=\"stylesheet\" href=\"style.css\" />\n")
	fmt.Fprintf(w, "<link rel=\"shortcut icon\" href=\"myself.png\" type=\"image/x-icon\">\n")
	fmt.Fprintf(w, "<script src=\"script.js\"></script>\n\n")

	fmt.Fprintf(w, "</head>\n<body>\n")
	fmt.Fprintf(w, "<h1 class=\"blog-title\">Hirrolot's Blog</h1>\n\n")
	err = appendFile(w, "badges.html")
	if err != nil {
		return err
	}

	err = genPostsHistory(w, postNames)
	if err != nil {
		return nil
	}

	fmt.Fprintln(w, "</body>\n</html>")
	return nil
}

func genPostsHistory(w io.Writer, postNames []string) error {
	meta, err := collectPostsMetadata(postNames)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "<div class=\"posts-history\">\n")

	minYear, maxYear := minPostYear(meta), maxPostYear(meta)

	for year := maxYear; year >= minYear; year-- {
		for month := time.December; month >= time.January; month-- {
			for day := 31; day >= 1; day-- {
				for i, post := range meta {
					if post.date.year != year || post.date.month != month || post.date.day != uint8(day) {
						continue
					}

					fmt.Fprintf(w,
						"<div class=\"post-link\"><a href=\" %s/%s.html\">%s</a><br><span class=\"post-date\">%s %d, %d</span></div>\n",
						outputDir, postNames[i], post.title,
						strMonth(post.date.month),
						post.date.day, post.date.year)
				}
			}
		}
	}

	fmt.Fprintln(w, "</div>") // class="posts-history"
	return nil
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

func collectPostNames() (posts []string, e error) {
	postNames, err := ioutil.ReadDir(contentDir)
	if err != nil {
		return nil, err
	}

	for _, post := range postNames {
		posts = append(posts, strings.TrimSuffix(post.Name(), filepath.Ext(post.Name())))
	}

	return posts, nil
}

func invokePandoc(postName string) error {
	cmd := exec.Command(
		"pandoc", fmt.Sprintf("%s/%s.md", contentDir, postName),
		"--output", fmt.Sprintf("%s/%s.html", outputDir, postName),
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
		return err
	}

	return nil
}

type PostMetadata struct {
	title string
	date  PostDate
}

type PostDate struct {
	day   uint8
	month time.Month
	year  uint32
}

func collectPostsMetadata(postNames []string) (meta []PostMetadata, e error) {
	for _, postName := range postNames {
		content, err := os.ReadFile(fmt.Sprintf("%s/%s.md", contentDir, postName))
		if err != nil {
			return nil, err
		}

		pandocMetadata := strings.Split(string(content), "---")[1]

		var postMeta PostMetadata
		for _, line := range strings.Split(pandocMetadata, "\n") {
			if strings.HasPrefix(line, "title: ") {
				postMeta.title = strings.TrimPrefix(line, "title: ")
			}
			if strings.HasPrefix(line, "date: ") {
				date := strings.TrimPrefix(line, "date: ")

				var month string
				_, err := fmt.Sscanf(date, "%s %d, %d",
					&month, &postMeta.date.day, &postMeta.date.year)
				if err != nil {
					return nil, err
				}

				postMeta.date.month = parseMonth(month)
			}
		}

		meta = append(meta, postMeta)
	}

	return meta, nil
}

func parseMonth(month string) time.Month {
	switch month {
	case "Jan":
		return time.January
	case "Feb":
		return time.February
	case "Mar":
		return time.March
	case "Apr":
		return time.April
	case "May":
		return time.May
	case "Jun":
		return time.June
	case "Jul":
		return time.July
	case "Aug":
		return time.August
	case "Sept":
		return time.September
	case "Oct":
		return time.October
	case "Nov":
		return time.November
	case "Dec":
		return time.December
	default:
		panic("No such month")
	}
}

func strMonth(month time.Month) string {
	switch month {
	case time.January:
		return "Jan"
	case time.February:
		return "Feb"
	case time.March:
		return "Mar"
	case time.April:
		return "Apr"
	case time.May:
		return "May"
	case time.June:
		return "Jun"
	case time.July:
		return "Jul"
	case time.August:
		return "Aug"
	case time.September:
		return "Sept"
	case time.October:
		return "Oct"
	case time.November:
		return "Nov"
	case time.December:
		return "Dec"
	default:
		panic("No such month")
	}
}

func minPostYear(meta []PostMetadata) uint32 {
	result := uint32(9999)
	for _, post := range meta {
		if result > post.date.year {
			result = post.date.year
		}
	}

	return result
}

func maxPostYear(meta []PostMetadata) uint32 {
	result := uint32(9999)
	for _, post := range meta {
		if result < post.date.year {
			result = post.date.year
		}
	}

	return result
}
