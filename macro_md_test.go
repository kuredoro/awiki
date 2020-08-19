package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestMMDConvert(t *testing.T) {
    cases := []struct{
        style MMDStyle
        input string
        want string
    } {
        {
            style: MMDStyle {
                "i": "*",
            },
            input: "test .i text",
            want: "*test* text",
        },
        {
            style: MMDStyle {
                "i": "*",
            },
            input: "test.i text .i",
            want: "*test* *text*",
        },
        {
            style: MMDStyle {
                "i": "*",
            },
            input: ".i.itest.i text .i",
            want: "*.itest* *text*",
        },
        {
            style: MMDStyle {
                "i": "*",
                "c": "`",
            },
            input: ".i.itest.c text .i",
            want: "`.itest` *text*",
        },
        {
            style: MMDStyle {
                "i": "*",
                "c": "`",
            },
            input: "test text .i2 is wierd sometimes .c3.",
            want: "*test text* `is wierd sometimes`.",
        },
    }

    for _, test := range cases {
        t.Run(test.input,
        func(t *testing.T) {
            mmd := NewMacroMarkdown(test.style)
            got := string(mmd.Convert([]byte(test.input)))

            if got != test.want {
                t.Errorf("got %q, want %q", got, test.want)
            }
        })
    }
}

func TestMMDFindMacros(t *testing.T) {
    style := MMDStyle {
        "i": "*",
        "ii": "!!",
        "abc": "/+",
    }

    cases := []struct {
        input []byte
        want []InTextMacro
    } {
        {
            input: []byte(".i"),
            want: []InTextMacro{ {"i", 1, 0, 2} },
        },
        {
            input: []byte(" .i "),
            want: []InTextMacro{ {"i", 1, 1, 3} },
        },
        {
            input: []byte(" .i.ii"),
            want: []InTextMacro{ {"i", 1, 1, 3}, {"ii", 1, 3, 6} },
        },
        {
            input: []byte(" .b.ii."),
            want: []InTextMacro{ {"ii", 1, 3, 6} },
        },
        {
            input: []byte(".abc3"),
            want: []InTextMacro{ {"abc", 3, 0, 5} },
        },
        {
            input: []byte("abc.abc30.i0 .ii5.ii.7."),
            want: []InTextMacro{ {"abc", 30, 3, 9}, {"i", 0, 9, 12}, {"ii", 5, 13, 17}, {"ii", 1, 17, 20} },
        },
    }

    for _, test := range cases {
        t.Run(string(test.input),
        func(t *testing.T) {
            mmd := NewMacroMarkdown(style)
            got := mmd.findMacroEntries(test.input)

            if !reflect.DeepEqual(got, test.want) {
                t.Errorf("got %v, want %v", got, test.want)
            }
        })
    }
}

func TestMMDSeekBack(t *testing.T) {
    text := []byte("This is an example.")
    cases := []struct {
        desc string
        pos int
        want int
    } {
        {
            pos: 8,
            want: 5,
        },
        {
            pos: 5,
            want: 0,
        },
        {
            pos: 0,
            want: 0,
        },
        {
            pos: 2,
            want: 0,
        },
        {
            pos: 6,
            want: 5,
        },
    }

    for _, test := range cases {
        t.Run(showOffsets(text, test.pos, -1),
        func(t *testing.T) {
            got := seekWordBackwards(text, test.pos)

            if got != test.want {
                info := showOffsets(text, test.want, got)
                t.Errorf("<got> %d, (want) %d, %s", got, test.want, info)
            }
        })
    }
}

func showOffsets(text []byte, wantPos, gotPos int) string {
    var str strings.Builder

    for i := range text {
        if i == wantPos {
            str.WriteByte('(')
            str.WriteByte(text[i])
            str.WriteByte(')')
        } else if i == gotPos {
            str.WriteByte('<')
            str.WriteByte(text[i])
            str.WriteByte('>')
        } else {
            str.WriteByte(text[i])
        }
    }

    return str.String()
}
