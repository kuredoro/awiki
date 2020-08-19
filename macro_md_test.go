package main

import (
	"reflect"
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
