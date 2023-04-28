package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
)

func resolveURL(url string, location string, root string) string {
	if strings.HasPrefix(url, "s") {
		return filepath.Join(root, url)
	}
	return filepath.Join(location, url)
}

func BuildFile(path string, projectRoot string, outDir string) {
	os.MkdirAll(outDir, 0777)

	file, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	doc, err := html.Parse(bytes.NewBuffer(file))
	if err != nil {
		panic(err)
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			var inlineIsTS = false
			var done = false
			for i, a := range n.Attr {
				if a.Key == "src" {
					fmt.Println("external")
					done = true
					outFileName := strings.Replace(filepath.Base(a.Val), filepath.Ext(a.Val), "", 1) + ".out.js"

					_, err := os.Stat(filepath.Join(outDir, outFileName))
					if os.IsNotExist(err) {
						result := api.Build(api.BuildOptions{
							EntryPoints: []string{resolveURL(a.Val, filepath.Dir(path), projectRoot)},
							Bundle:      true,
							Write:       true,
							Outfile:     filepath.Join(outDir, outFileName),
						})
						if len(result.Errors) != 0 {
							fmt.Println(result.Errors)
							os.Exit(1)
						}
					}

					n.Attr[i].Val, err = filepath.Rel(outDir, filepath.Join(outDir, outFileName))
					if err != nil {
						panic(err)
					}
					fmt.Println(a.Key, a.Val)

				} else if a.Key == "type" && a.Val == "ts" {
					inlineIsTS = true
				}

			}
			fmt.Println(n.Attr)
			if !done {

				fileName := "tmp.js"
				if inlineIsTS {
					fileName = "tmp.ts"
				}
				fmt.Println(filepath.Join(outDir, fileName))
				err = os.WriteFile(filepath.Join(outDir, fileName), []byte(n.FirstChild.Data), 0777)
				if err != nil {
					panic(err)
				}
				result := api.Build(api.BuildOptions{
					EntryPoints: []string{filepath.Join(outDir, fileName)},
					Bundle:      true,
					Write:       true,
					Outfile:     filepath.Join(outDir, "tmp.out.js"),
				})
				if len(result.Errors) != 0 {
					fmt.Println(result.Errors)
					os.Exit(1)
				}
				f, err := os.ReadFile(filepath.Join(outDir, fileName))
				if err != nil {
					panic(err)
				}
				fmt.Println(n.Data)
				newAttr := []html.Attribute{}
				for _, a := range n.Attr {
					if a.Key != "type" {
						newAttr = append(newAttr, a)
					}
				}
				n.Attr = newAttr
				n.FirstChild.Data = html.UnescapeString(string(f))
				os.Remove(filepath.Join(outDir, fileName))
				os.Remove(filepath.Join(outDir, "tmp.out.js"))
			}

		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	fmt.Println("fefewdnfenf")

	out, err := os.Create(filepath.Join(outDir, "out.html"))
	if err != nil {
		panic(err)
	}
	html.Render(out, doc)
}
