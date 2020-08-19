package main

import (
	"log"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type MMDStyle map[string]string

var defaultMacros = MMDStyle{
    "i": "*",
    "b": "**",
    "c": "`",
    "m": "$",
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

// Convert replaces macro definitions to the corresponding enlosing
// markdown as specified in *style*. Unfortunately, stacking is not allowed,
// i.e., this won't work "bold italic .b2 .i2".
func (c *MMDConverter) Convert(text []byte) []byte {
    type event struct {
        pos int
        id int
        str string
    }

    // For each macro generate a pair of "events", i.e. positions
    // at which symbols need to be inserted.
    macros := c.findMacroEntries(text)
    var events []event

    for i, macro := range macros {
        mdStart := macro.byteBegin
        for d := 0; d < macro.degree; d++ {
            mdStart = seekWordBackwards(text, mdStart)
        }

        mdEnd := seekNonSpaceBackwards(text, macro.byteBegin) + 1

        if mdStart == mdEnd {
            continue
        }

        chars := c.style[macro.name]
        events = append(events, event{mdStart, -i - 1, chars})
        events = append(events, event{mdEnd, i + 1, chars})
    }

    sort.Slice(events, func(i, j int) bool {
        if events[i].pos == events[j].pos {
            return events[i].id < events[j].id
        }
        return events[i].pos < events[j].pos
    })

    var str strings.Builder

    ti, mi, ei := 0, 0, 0
    for ; ; ti++ {
        for ei < len(events) && events[ei].pos <= ti {
            str.WriteString(events[ei].str)
            ei++
        }

        for mi < len(macros) {
            skipBegin := seekNonSpaceBackwards(text, macros[mi].byteBegin) + 1
            if ti < skipBegin || macros[mi].byteEnd <= ti {
                break
            }

            ti = macros[mi].byteEnd
            mi++
        }

        if ti >= len(text) {
            break
        }

        str.WriteByte(text[ti])
    }

    return []byte(str.String())
}

// findMacroEntries will search for the .<name><number> sequences that are known 
// macro definitions (as defined in *style*). They will be listed in order of their
// appearance.
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
                    // This is an exotic error, I don't wanna bother...
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

// seekWordBackwards will return the position of the "cursor" as if after b
// was pressed in vim in normal mode. In other words, jump to the beginnig of the
// previous word, or the text otherwise.
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

// seekNonSpaceBackwards will return the index of the first non-space character
// before the current position. If there's no such character, -1 is returned.
func seekNonSpaceBackwards(text []byte, pos int) int {
    pos--
    for ; pos >= 0; pos-- {
        if !unicode.IsSpace(rune(text[pos])) {
            break
        }
    }

    return pos
}
