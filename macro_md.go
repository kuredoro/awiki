package main

import (
	"strconv"
    "log"
	"unicode"
)

type MMDStyle map[string]string

var defaultMacros = MMDStyle{

}

type InTextMacro struct {
    name string
    degree int

    byteBegin int
    byteEnd int
}

type MMDConverter struct {
    style MMDStyle
}

func NewMacroMarkdown(style MMDStyle) *MMDConverter {
    return &MMDConverter{style}
}

func (c *MMDConverter) Convert(text []byte) []byte {
    
    return []byte("*test* text")
}

func (c *MMDConverter) findMacroEntries(text []byte) (macros []InTextMacro) {
    for i := 0; i < len(text); i++ {
        if text[i] == '.' {
            nameEnd := i + 1
            for ; nameEnd < len(text); nameEnd++ {
                if !unicode.IsLetter(rune(text[nameEnd])) {
                    break
                }
            }

            name := string(text[i+1:nameEnd])

            _, knownName := c.style[name]
            if !knownName {
                continue
            }

            degreeEnd := nameEnd
            for ; degreeEnd < len(text); degreeEnd++ {
                if !unicode.IsDigit(rune(text[degreeEnd])) {
                    break
                }
            }

            degree := 1
            if degreeEnd != nameEnd {
                degreeStr := string(text[nameEnd:degreeEnd])

                var err error
                degree, err = strconv.Atoi(degreeStr)
                if err != nil {
                    log.Printf("Got known macro with invalid degree %q, %v; skipping...", degreeStr, err)
                    continue
                }
            }

            macro := InTextMacro{
                name: name, 
                degree: degree,
                byteBegin: i,
                byteEnd: degreeEnd,
            }

            macros = append(macros, macro)

            i = degreeEnd - 1
        }
    }

    return
}


func seekWordBackwards(text []byte, pos int) int {
    pos--

    for ; pos >= 0; pos-- {
        if !unicode.IsSpace(rune(text[pos])) {
            break
        }
    }

    for ; pos >= 0; pos-- {
        if unicode.IsSpace(rune(text[pos])) {
            break
        }
    }

    return pos+1
}
